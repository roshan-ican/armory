package services

import (
	"context"
	"errors"
	"log"
	"slices"
	"strings"
	"sync"

	"armory/internal/database"
	"armory/internal/live"
	"armory/internal/models"
)

const (
	recentRequests  = 30
	requestPageSize = 25
)

type RequestService struct {
	store *database.Store
	hw    Hardware
	hub   *live.Hub

	returnMu  sync.Mutex
	returning map[int64]string
}

const (
	ReturnPending  = "pending"
	ReturnApproved = "approved"
	ReturnDeclined = "declined"
)

func NewRequestService(store *database.Store, hw Hardware, hub *live.Hub) *RequestService {
	return &RequestService{store: store, hw: hw, hub: hub, returning: map[int64]string{}}
}

type ReturnView struct {
	Request  models.Request
	Taken    string
	Returned string
	StillOut string
	PutBack  string
}

func slotsWithStatus(slots []models.RequestSlot, statuses ...string) []int64 {
	var out []int64
	for _, rs := range slots {
		if slices.Contains(statuses, rs.Status) {
			out = append(out, rs.SlotNo)
		}
	}
	return out
}

func (s *RequestService) setReturnState(id int64, state string) {
	s.returnMu.Lock()
	defer s.returnMu.Unlock()
	if state == "" {
		delete(s.returning, id)
	} else {
		s.returning[id] = state
	}
}

func (s *RequestService) returnStateOf(id int64) string {
	s.returnMu.Lock()
	defer s.returnMu.Unlock()
	return s.returning[id]
}

func (s *RequestService) Returning(ctx context.Context) ([]ReturnView, error) {
	s.returnMu.Lock()
	ids := make([]int64, 0, len(s.returning))
	for id, state := range s.returning {
		if state == ReturnPending {
			ids = append(ids, id)
		}
	}
	s.returnMu.Unlock()
	slices.Sort(ids)

	var out []ReturnView
	for _, id := range ids {
		req, err := s.store.GetRequest(ctx, id)
		if err != nil || (req.Status != models.RequestApproved && req.Status != models.RequestCollected) {
			s.setReturnState(id, "")
			continue
		}
		wrong, err := s.store.WrongSlots(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, ReturnView{
			Request:  req,
			Taken:    slotWords(slotsWithStatus(req.Slots, models.SlotCollected, models.SlotReturned)),
			Returned: slotWords(slotsWithStatus(req.Slots, models.SlotReturned)),
			StillOut: slotWords(slotsWithStatus(req.Slots, models.SlotCollected)),
			PutBack:  slotWords(wrong),
		})
	}
	return out, nil
}

type RequestView struct {
	Request     models.Request
	Slots       []models.Slot
	Wrong       []int64
	ReturnState string
}

type CatalogSlot struct {
	No          int64  `json:"no"`
	Reading     int64  `json:"reading"`
	TakenBy     string `json:"taken_by"`
	RequestedBy string `json:"requested_by"`
	Available   bool   `json:"available"`
	Disabled    bool   `json:"disabled"`
}

type CatalogItem struct {
	ID        int64         `json:"id"`
	Name      string        `json:"name"`
	Location  string        `json:"location"`
	Kind      string        `json:"kind"`
	Capacity  int64         `json:"capacity"`
	Online    bool          `json:"online"`
	Available int64         `json:"available"`
	Slots     []CatalogSlot `json:"slots"`
}

func translateRequestError(err error) error {
	switch {
	case errors.Is(err, database.ErrNotFound):
		return ErrRequestNotFound
	case errors.Is(err, database.ErrConflict):
		return ErrRequestClosed
	case errors.Is(err, database.ErrNoFreeSlot):
		return ErrNoneAvailable
	}
	return err
}

func (s *RequestService) guardMaintenance(ctx context.Context) error {
	active, err := s.store.MaintenanceActive(ctx)
	if err != nil {
		return err
	}
	if active {
		return ErrMaintenanceActive
	}
	return nil
}

func (s *RequestService) Create(ctx context.Context, userID int64, kind, reason string) (models.Request, error) {
	if err := s.guardMaintenance(ctx); err != nil {
		return models.Request{}, err
	}
	if !models.ValidKind(kind) {
		return models.Request{}, ErrInvalidKind
	}
	if _, err := s.store.GetUser(ctx, userID); errors.Is(err, database.ErrNotFound) {
		return models.Request{}, ErrUserNotFound
	} else if err != nil {
		return models.Request{}, err
	}

	open, err := s.store.HasOpenRequest(ctx, userID)
	if err != nil {
		return models.Request{}, err
	}
	if open {
		return models.Request{}, ErrRequestOpen
	}

	id, err := s.store.CreateRequest(ctx, userID, kind, strings.TrimSpace(reason), "")
	if err != nil {
		return models.Request{}, err
	}
	log.Printf("request %d: user %d asked for a %s", id, userID, kind)
	s.hub.Publish()
	return s.store.GetRequest(ctx, id)
}

func (s *RequestService) CreateForLocker(ctx context.Context, userID, lockerID int64, slotNos []int64, reason string) (models.Request, error) {
	if err := s.guardMaintenance(ctx); err != nil {
		return models.Request{}, err
	}
	locker, err := s.store.GetLocker(ctx, lockerID)
	if errors.Is(err, database.ErrNotFound) {
		return models.Request{}, ErrNoneAvailable
	}
	if err != nil {
		return models.Request{}, err
	}
	slotNos, err = cleanSlotNos(slotNos, locker.Capacity)
	if err != nil {
		return models.Request{}, err
	}
	if n, err := s.store.CountAvailableInLocker(ctx, lockerID); err != nil {
		return models.Request{}, err
	} else if n == 0 {
		return models.Request{}, ErrNoneAvailable
	}
	if _, err := s.store.GetUser(ctx, userID); errors.Is(err, database.ErrNotFound) {
		return models.Request{}, ErrUserNotFound
	} else if err != nil {
		return models.Request{}, err
	}
	if open, err := s.store.HasOpenRequest(ctx, userID); err != nil {
		return models.Request{}, err
	} else if open {
		return models.Request{}, ErrRequestOpen
	}
	id, err := s.store.CreateRequestForLocker(ctx, userID, lockerID, locker.Kind, strings.TrimSpace(reason), "", slotNos)
	if errors.Is(err, database.ErrNoFreeSlot) {
		return models.Request{}, ErrGunUnavailable
	}
	if err != nil {
		return models.Request{}, err
	}
	log.Printf("request %d: user %d asked for %s slots %v (%s)", id, userID, locker.Name, slotNos, locker.Kind)
	s.hub.Publish()
	return s.store.GetRequest(ctx, id)
}

func cleanSlotNos(in []int64, capacity int64) ([]int64, error) {
	seen := make(map[int64]bool, len(in))
	out := make([]int64, 0, len(in))
	for _, no := range in {
		if no < 1 || no > capacity {
			return nil, ErrInvalidSlot
		}
		if !seen[no] {
			seen[no] = true
			out = append(out, no)
		}
	}
	slices.Sort(out)
	return out, nil
}

func (s *RequestService) Catalog(ctx context.Context) ([]CatalogItem, error) {
	if err := s.guardMaintenance(ctx); err != nil {
		return nil, err
	}
	lockers, err := s.store.ListLockers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CatalogItem, 0, len(lockers))
	for _, locker := range lockers {
		n, err := s.store.CountAvailableInLocker(ctx, locker.ID)
		if err != nil {
			return nil, err
		}
		slots := make([]CatalogSlot, 0, len(locker.Slots))
		for _, sl := range locker.Slots {
			slots = append(slots, CatalogSlot{
				No: sl.SlotNo, Reading: sl.Reading, TakenBy: sl.TakenBy, RequestedBy: sl.RequestedBy,
				Available: locker.Online && sl.Reading == 1 && sl.TakenBy == "",
				Disabled:  sl.SensorDisabled,
			})
		}
		out = append(out, CatalogItem{
			ID: locker.ID, Name: locker.Name, Location: locker.Location, Kind: locker.Kind,
			Capacity: locker.Capacity, Online: locker.Online, Available: n, Slots: slots,
		})
	}
	return out, nil
}

func (s *RequestService) Approve(ctx context.Context, adminID, id int64) (models.Request, error) {
	if err := requireAdmin(ctx, s.store, adminID); err != nil {
		return models.Request{}, err
	}
	req, err := s.store.ApproveRequest(ctx, id, adminID)
	if err != nil {
		log.Printf("request %d: approve by admin %d failed: %v", id, adminID, err)
		if errors.Is(err, database.ErrNoFreeSlot) {
			if pending, gerr := s.store.GetRequest(ctx, id); gerr == nil && len(pending.Slots) > 0 {
				return models.Request{}, ErrGunUnavailable
			}
		}
		return models.Request{}, translateRequestError(err)
	}
	nos := req.SlotNos()
	log.Printf("request %d: approved by admin %d, %s slots %v (%s)", id, adminID, req.LockerName, nos, req.LockerIP)
	if len(nos) > 0 {
		if err := s.hw.Unlock(ctx, req.LockerIP, nos[0]); err != nil {
			log.Printf("unlock locker %s: %v", req.LockerIP, err)
		}
	}
	doorOpened := true
	if err := s.hw.OpenDoor(ctx); err != nil {
		log.Printf("open door : %v", err)
		doorOpened = false
	}
	s.hub.Say(ApprovedMessage(req.LockerName, nos, doorOpened))
	s.hub.Publish()
	return req, nil
}

func (s *RequestService) Reject(ctx context.Context, adminID, id int64) error {
	if err := requireAdmin(ctx, s.store, adminID); err != nil {
		return err
	}
	if err := s.store.RejectRequest(ctx, id, adminID, ""); err != nil {
		log.Printf("request %d: reject by admin %d failed: %v", id, adminID, err)
		return translateRequestError(err)
	}
	log.Printf("request %d: declined by admin %d", id, adminID)
	s.hub.Publish()
	return nil
}

func (s *RequestService) Cancel(ctx context.Context, userID, id int64) error {
	if err := s.store.CancelRequest(ctx, id, userID); err != nil {
		log.Printf("request %d: cancel by user %d failed: %v", id, userID, err)
		return translateRequestError(err)
	}
	log.Printf("request %d: cancelled by user %d", id, userID)
	s.hub.Publish()
	return nil
}

func (s *RequestService) Close(ctx context.Context, actorID, id int64, force bool) ([]string, error) {
	actor, err := s.store.GetUser(ctx, actorID)
	if errors.Is(err, database.ErrNotFound) {
		return nil, ErrNotAdmin
	}
	if err != nil {
		return nil, err
	}
	admin := actor.IsAdmin()
	if force && !admin {
		return nil, ErrNotAdmin
	}

	req, err := s.store.GetRequest(ctx, id)
	if err != nil {
		return nil, translateRequestError(err)
	}
	if !admin && req.UserID != actorID {
		return nil, ErrRequestNotFound
	}
	if req.Status != models.RequestReturned || req.ClosedAt != "" {
		return nil, ErrRequestClosed
	}

	slots, err := s.store.SlotsOfLocker(ctx, req.LockerID)
	if err != nil {
		return nil, err
	}
	wrong, err := s.store.WrongSlots(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	problems := VerifyReturn(req, slots, wrong)

	note := ""
	if len(problems) > 0 {
		note = strings.Join(problems, " ")
		if !force {
			if err := s.store.FlagRequest(ctx, id, note); err != nil {
				return nil, err
			}
			log.Printf("request %d: close by %s refused, %d problems", id, actor.Name, len(problems))
			s.hub.Say(CloseProblemsMessage(problems))
			s.hub.Publish()
			return problems, nil
		}
		note = "closed by " + actor.Name + " despite: " + note
	}

	if err := s.store.CloseRequest(ctx, id, note); err != nil {
		return nil, translateRequestError(err)
	}
	s.setReturnState(id, "")
	log.Printf("request %d: closed by %s", id, actor.Name)
	s.hub.Say(ClosedMessage())
	s.hub.Publish()
	return nil, nil
}

func (s *RequestService) Pending(ctx context.Context) ([]models.Request, error) {
	return s.store.ListPendingRequests(ctx)
}

func (s *RequestService) AwaitingClose(ctx context.Context) ([]models.Request, error) {
	return s.store.ListAwaitingClose(ctx)
}

func (s *RequestService) Recent(ctx context.Context) ([]models.Request, error) {
	return s.store.ListRecentRequests(ctx, recentRequests)
}

type RequestHistoryPage struct {
	Pager
	Requests []models.Request
}

func (s *RequestService) HistoryPage(ctx context.Context, page int) (RequestHistoryPage, error) {
	total, err := s.store.CountRequestHistory(ctx)
	if err != nil {
		return RequestHistoryPage{}, err
	}
	pager := newPager(total, requestPageSize, page)
	list, err := s.store.ListRequestHistory(ctx, requestPageSize, pager.offset())
	if err != nil {
		return RequestHistoryPage{}, err
	}
	pager.shown(len(list))
	return RequestHistoryPage{Pager: pager, Requests: list}, nil
}

func (s *RequestService) view(ctx context.Context, req models.Request) (RequestView, error) {
	v := RequestView{Request: req, ReturnState: s.returnStateOf(req.ID)}
	awaitingClose := req.Status == models.RequestReturned && req.ClosedAt == ""
	if req.Status != models.RequestApproved && req.Status != models.RequestCollected && !awaitingClose {
		return v, nil
	}
	slots, err := s.store.SlotsOfLocker(ctx, req.LockerID)
	if err != nil {
		return RequestView{}, err
	}
	v.Slots = slots
	wrong, err := s.store.WrongSlots(ctx, req.ID)
	if err != nil {
		return RequestView{}, err
	}
	v.Wrong = wrong
	return v, nil
}

func (s *RequestService) View(ctx context.Context, userID, id int64) (RequestView, error) {
	req, err := s.store.GetRequest(ctx, id)
	if err != nil {
		return RequestView{}, translateRequestError(err)
	}
	if req.UserID != userID {
		return RequestView{}, ErrRequestNotFound
	}
	return s.view(ctx, req)
}

func (s *RequestService) Open(ctx context.Context, userID int64) (RequestView, error) {
	req, err := s.store.LatestOpenRequest(ctx, userID)
	if err != nil {
		return RequestView{}, translateRequestError(err)
	}
	return s.view(ctx, req)
}

func (s *RequestService) Reopen(ctx context.Context, userID int64) (RequestView, error) {
	v, err := s.Open(ctx, userID)
	if err != nil || v.Request.Status != models.RequestApproved {
		return v, err
	}
	reading := make(map[int64]int64, len(v.Slots))
	for _, sl := range v.Slots {
		reading[sl.SlotNo] = sl.Reading
	}
	for _, rs := range v.Request.Slots {
		if rs.Status == models.SlotChosen && reading[rs.SlotNo] == 1 {
			log.Printf("request %d: gun %d still in %s, opening again", v.Request.ID, rs.SlotNo, v.Request.LockerName)
			if err := s.hw.Unlock(ctx, v.Request.LockerIP, rs.SlotNo); err != nil {
				log.Printf("unlock locker %s: %v", v.Request.LockerIP, err)
			}
			if err := s.hw.OpenDoor(ctx); err != nil {
				log.Printf("open door: %v", err)
			}
			break
		}
	}
	return v, nil
}

func (s *RequestService) OpenForReturn(ctx context.Context, userID int64) error {
	v, err := s.Open(ctx, userID)
	if err != nil {
		return err
	}
	switch v.Request.Status {
	case models.RequestApproved, models.RequestCollected, models.RequestReturned:
	default:
		return ErrRequestClosed
	}
	needsApproval := v.Request.Status != models.RequestApproved && len(v.Wrong) == 0 && v.ReturnState != ReturnApproved
	if !needsApproval {
		s.unlockForReturn(ctx, v)
		log.Printf("request %d: user %d opened %s to return guns", v.Request.ID, userID, v.Request.LockerName)
		return nil
	}
	if v.ReturnState != ReturnPending {
		s.setReturnState(v.Request.ID, ReturnPending)
		s.hub.Notice(AdminReturnRequestMessage(
			v.Request.UserName,
			slotsWithStatus(v.Request.Slots, models.SlotCollected, models.SlotReturned),
			slotsWithStatus(v.Request.Slots, models.SlotCollected),
		))
		log.Printf("request %d: user %d asked to return guns, waiting for admin", v.Request.ID, userID)
		s.hub.Publish()
	}
	return nil
}

func (s *RequestService) unlockForReturn(ctx context.Context, v RequestView) {
	nos := slotsWithStatus(v.Request.Slots, models.SlotCollected)
	nos = append(nos, v.Wrong...)
	for _, no := range nos {
		if err := s.hw.Unlock(ctx, v.Request.LockerIP, no); err != nil {
			log.Printf("unlock locker %s: %v", v.Request.LockerIP, err)
		}
	}
	if err := s.hw.OpenDoor(ctx); err != nil {
		log.Printf("open door: %v", err)
	}
}

func (s *RequestService) pendingReturn(ctx context.Context, adminID, id int64) (RequestView, error) {
	if err := requireAdmin(ctx, s.store, adminID); err != nil {
		return RequestView{}, err
	}
	req, err := s.store.GetRequest(ctx, id)
	if err != nil {
		return RequestView{}, translateRequestError(err)
	}
	if s.returnStateOf(id) != ReturnPending {
		return RequestView{}, ErrRequestClosed
	}
	return s.view(ctx, req)
}

func (s *RequestService) ApproveReturn(ctx context.Context, adminID, id int64) error {
	v, err := s.pendingReturn(ctx, adminID, id)
	if err != nil {
		return err
	}
	s.setReturnState(id, ReturnApproved)
	s.unlockForReturn(ctx, v)
	log.Printf("request %d: admin %d approved the return", id, adminID)
	s.hub.Say(ReturnApprovedMessage(v.Request.LockerName))
	s.hub.Publish()
	return nil
}

func (s *RequestService) DeclineReturn(ctx context.Context, adminID, id int64) error {
	if _, err := s.pendingReturn(ctx, adminID, id); err != nil {
		return err
	}
	s.setReturnState(id, ReturnDeclined)
	log.Printf("request %d: admin %d declined the return", id, adminID)
	s.hub.Say(ReturnDeclinedMessage())
	s.hub.Publish()
	return nil
}

func (s *RequestService) Availability(ctx context.Context) (map[string]int64, error) {
	out := make(map[string]int64, 2)
	for _, kind := range models.Kinds {
		n, err := s.store.CountAvailable(ctx, kind)
		if err != nil {
			return nil, err
		}
		out[kind] = n
	}
	return out, nil
}

func (s *RequestService) SlotChanged(ctx context.Context, slotID int64, reading byte) {
	s.handleSlot(ctx, slotID, reading)
	s.refreshAlarm(ctx, slotID)
}

func (s *RequestService) refreshAlarm(ctx context.Context, slotID int64) {
	info, err := s.store.GetSlotInfo(ctx, slotID)
	if err != nil {
		return
	}
	ip, err := s.store.LockerIPOfSlot(ctx, slotID)
	if err != nil {
		return
	}
	s.RefreshLockerAlarm(ctx, info.LockerID, ip)
}

func (s *RequestService) RefreshLockerAlarm(ctx context.Context, lockerID int64, ip string) {
	if ip == "" {
		return
	}
	on := false
	if approved, err := s.store.ApprovedRequestInLocker(ctx, lockerID); err == nil {
		wrong, err := s.store.WrongSlots(ctx, approved.ID)
		on = err == nil && len(wrong) > 0
	}
	if !on {
		if returned, err := s.store.UnclosedReturnedRequestInLocker(ctx, lockerID); err == nil {
			wrong, err := s.store.WrongSlots(ctx, returned.ID)
			on = err == nil && len(wrong) > 0
		}
	}
	if !on {
		unauthorized, err := s.store.HasUnauthorizedEmpty(ctx, lockerID)
		on = err == nil && unauthorized
	}
	if err := s.hw.Alarm(ctx, ip, on); err != nil {
		log.Printf("alarm %t to %s: %v", on, ip, err)
	}
}

func (s *RequestService) handleSlot(ctx context.Context, slotID int64, reading byte) {
	switch reading {
	case 0:
		id, done, err := s.store.MarkCollected(ctx, slotID)
		if err != nil {
			log.Printf("mark collected slot %d: %v", slotID, err)
			return
		}
		if id != 0 {
			log.Printf("request %d: slot %d emptied, a chosen gun was taken (all taken: %t)", id, slotID, done)
			s.signalSlot(ctx, slotID, SignalCorrect)
			s.announceCorrect(ctx, id, slotID, done)
			s.hub.Publish()
			return
		}
		info, err := s.store.GetSlotInfo(ctx, slotID)
		if err != nil {
			return
		}
		approved, err := s.store.ApprovedRequestInLocker(ctx, info.LockerID)
		if err != nil {
			s.flagLooseTake(ctx, slotID, info)
			return
		}
		log.Printf("request %d: locker %d slot %d emptied but it was not chosen, wrong gun", approved.ID, info.LockerID, info.SlotNo)
		if err := s.store.RecordWrongGun(ctx, slotID, approved.ID); err != nil {
			log.Printf("record wrong gun slot %d: %v", slotID, err)
			return
		}
		pending := pendingSlotNumbers(approved.Slots)
		s.hub.Say(WrongPickMessage(info.SlotNo, pending))
		s.hub.Alert(AdminWrongPickMessage(approved.UserName, info.SlotNo, pending))
		s.hub.Publish()
	case 1:
		id, done, err := s.store.MarkReturned(ctx, slotID)
		if err != nil {
			log.Printf("mark returned slot %d: %v", slotID, err)
			return
		}
		if id != 0 {
			log.Printf("request %d: slot %d is back (all back: %t)", id, slotID, done)
			if done {
				s.hub.Say(ReturnedMessage())
			}
			s.announceReturned(ctx, id, slotID, done)
			s.hub.Publish()
		}
	}
}

func (s *RequestService) flagLooseTake(ctx context.Context, slotID int64, info database.SlotInfo) {
	if active, err := s.store.MaintenanceActive(ctx); err != nil || active {
		return
	}
	if returned, err := s.store.UnclosedReturnedRequestInLocker(ctx, info.LockerID); err == nil {
		if err := s.store.RecordWrongGun(ctx, slotID, returned.ID); err != nil {
			log.Printf("record take after return slot %d: %v", slotID, err)
			return
		}
		log.Printf("request %d: locker %d slot %d emptied after the request was returned", returned.ID, info.LockerID, info.SlotNo)
		s.hub.Say(TakenAfterReturnMessage(info.SlotNo))
		s.hub.Alert(AdminTakenAfterReturnMessage(returned.UserName, info.SlotNo))
		s.hub.Publish()
		return
	}
	if err := s.store.RecordUnauthorizedTake(ctx, slotID); err != nil {
		log.Printf("record unauthorized take slot %d: %v", slotID, err)
		return
	}
	log.Printf("locker %d slot %d: gun left with no approved request", info.LockerID, info.SlotNo)
	s.hub.Say(UnauthorizedTakeMessage(info.SlotNo))
	s.hub.Alert(AdminUnauthorizedTakeMessage(info.SlotNo))
	s.hub.Publish()
}

func (s *RequestService) announceCorrect(ctx context.Context, requestID, slotID int64, last bool) {
	info, err := s.store.GetSlotInfo(ctx, slotID)
	if err != nil {
		return
	}
	s.hub.Say(CorrectPickMessage(info.SlotNo, last))
	s.hub.Notice(AdminCorrectPickMessage(s.requesterName(ctx, requestID), info.SlotNo, last))
}

func (s *RequestService) announceReturned(ctx context.Context, requestID, slotID int64, last bool) {
	info, err := s.store.GetSlotInfo(ctx, slotID)
	if err != nil {
		return
	}
	s.hub.Notice(AdminReturnedMessage(s.requesterName(ctx, requestID), info.SlotNo, last))
}

func (s *RequestService) requesterName(ctx context.Context, requestID int64) string {
	req, err := s.store.GetRequest(ctx, requestID)
	if err != nil {
		return ""
	}
	return req.UserName
}

func (s *RequestService) signalSlot(ctx context.Context, slotID int64, signal Signal) {
	ip, err := s.store.LockerIPOfSlot(ctx, slotID)
	if err != nil {
		log.Printf("signal %s: no locker IP for slot %d: %v", signal, slotID, err)
		return
	}
	if err := s.hw.Signal(ctx, ip, signal); err != nil {
		log.Printf("signal %s to %s: %v", signal, ip, err)
	}
}
