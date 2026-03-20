package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// CollectionTicketInventory is the collection name from the project outline.
const CollectionTicketInventory = "ticket_inventory"

// InventoryRepository persists ticket inventory documents.
type InventoryRepository struct {
	coll *mongo.Collection
}

// NewInventoryRepository builds a repository for ticket_inventory.
func NewInventoryRepository(db *mongo.Database) *InventoryRepository {
	return &InventoryRepository{
		coll: db.Collection(CollectionTicketInventory),
	}
}

// Ping checks connectivity to MongoDB.
func (r *InventoryRepository) Ping(ctx context.Context) error {
	return r.coll.Database().Client().Ping(ctx, nil)
}
