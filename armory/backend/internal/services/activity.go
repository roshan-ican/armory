package services

import (
	"context"
	"net/url"

	"armory/internal/database"
	"armory/internal/models"
)

const (
	activityLimit    = 200
	activityPageSize = 25
)

type ActivityFilter = database.EventFilter

type ActivityPage struct {
	Pager
	Events  []models.Event
	Filter  ActivityFilter
	Query   string
	People  []string
	Lockers []string
	Types   []ActivityType
}

type ActivityType struct {
	Value string
	Label string
}

var activityTypes = []ActivityType{
	{"gun_removed", "Taken"},
	{"gun_returned", "Returned"},
	{"wrong_gun", "Wrong gun"},
	{"unauthorized_take", "Taken without a request"},
	{"request_expired", "Not collected"},
	{"sensor_fault", "Sensor not detecting"},
	{"sensor_recovered", "Sensor detecting again"},
	{"maintenance_requested", "Maintenance requested"},
	{"maintenance_approved", "Maintenance approved"},
	{"maintenance_assigned", "Maintenance assigned"},
	{"maintenance_flagged", "Maintenance not as requested"},
	{"maintenance_started", "Maintenance started"},
	{"maintenance_finished", "Maintenance finished"},
	{"maintenance_cancelled", "Maintenance cancelled"},
}

type ActivityService struct {
	store *database.Store
}

func NewActivityService(store *database.Store) *ActivityService {
	return &ActivityService{store: store}
}

func (s *ActivityService) Recent(ctx context.Context) ([]models.Event, error) {
	return s.store.ListEvents(ctx, activityLimit)
}

func (s *ActivityService) Page(ctx context.Context, page int, f ActivityFilter) (ActivityPage, error) {
	total, err := s.store.CountEvents(ctx, f)
	if err != nil {
		return ActivityPage{}, err
	}
	pager := newPager(total, activityPageSize, page)
	events, err := s.store.ListEventsPage(ctx, f, activityPageSize, pager.offset())
	if err != nil {
		return ActivityPage{}, err
	}
	people, err := s.store.EventPeople(ctx)
	if err != nil {
		return ActivityPage{}, err
	}
	lockers, err := s.store.ListLockers(ctx)
	if err != nil {
		return ActivityPage{}, err
	}
	p := ActivityPage{
		Pager: pager, Events: events,
		Filter: f, Query: filterQuery(f), People: people, Types: activityTypes,
	}
	for _, l := range lockers {
		p.Lockers = append(p.Lockers, l.Name)
	}
	p.shown(len(events))
	return p, nil
}

func filterQuery(f ActivityFilter) string {
	v := url.Values{}
	for key, val := range map[string]string{"person": f.Person, "locker": f.Locker, "type": f.Type, "from": f.From, "to": f.To} {
		if val != "" {
			v.Set(key, val)
		}
	}
	return v.Encode()
}
