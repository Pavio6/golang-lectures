package items

import "sync"

type ItemMemoryRepository struct {
	mu     sync.RWMutex
	lastID uint32
	data   []*Item
}

var _ ItemsRepo = &ItemMemoryRepository{}

func NewMemoryRepo() *ItemMemoryRepository {
	return &ItemMemoryRepository{
		data: make([]*Item, 0, 10),
	}
}

func (repo *ItemMemoryRepository) GetAll() ([]*Item, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	items := make([]*Item, len(repo.data))
	for i, item := range repo.data {
		itemCopy := *item
		items[i] = &itemCopy
	}
	return items, nil
}

func (repo *ItemMemoryRepository) GetByID(id uint32) (*Item, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	for _, item := range repo.data {
		if item.ID == id {
			itemCopy := *item
			return &itemCopy, nil
		}
	}
	return nil, nil
}

func (repo *ItemMemoryRepository) Add(item *Item) (uint32, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	repo.lastID++
	item.ID = repo.lastID
	itemCopy := *item
	repo.data = append(repo.data, &itemCopy)
	return repo.lastID, nil
}

func (repo *ItemMemoryRepository) Update(newItem *Item) (bool, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	for _, item := range repo.data {
		if item.ID != newItem.ID {
			continue
		}
		item.Title = newItem.Title
		item.Description = newItem.Description
		return true, nil
	}
	return false, nil
}

func (repo *ItemMemoryRepository) Delete(id uint32) (bool, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	i := -1
	for idx, item := range repo.data {
		if item.ID != id {
			continue
		}
		i = idx
	}
	if i < 0 {
		return false, nil
	}

	if i < len(repo.data)-1 {
		copy(repo.data[i:], repo.data[i+1:])
	}
	repo.data[len(repo.data)-1] = nil // or the zero value of T
	repo.data = repo.data[:len(repo.data)-1]

	return true, nil
}
