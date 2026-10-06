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
	Events  []models.Event
	Page    int
	Pages   int
	Total   int
	First   int
	Last    int
	HasPrev bool
	HasNext bool
	Prev    int
	Next    int
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
	{"request_expired", "Not collected"},
	{"sensor_fault", "Sensor not detecting"},
	{"sensor_recovered", "Sensor detecting again"},
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
	pages := (total + activityPageSize - 1) / activityPageSize
	if pages < 1 {
		pages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}
	events, err := s.store.ListEventsPage(ctx, f, activityPageSize, (page-1)*activityPageSize)
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
		Events: events, Page: page, Pages: pages, Total: total,
		HasPrev: page > 1, HasNext: page < pages, Prev: page - 1, Next: page + 1,
		Filter: f, Query: filterQuery(f), People: people, Types: activityTypes,
	}
	for _, l := range lockers {
		p.Lockers = append(p.Lockers, l.Name)
	}
	if total > 0 {
		p.First = (page-1)*activityPageSize + 1
		p.Last = p.First + len(events) - 1
	}
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
