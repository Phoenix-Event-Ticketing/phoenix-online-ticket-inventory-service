package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CollectionTicketInventory is the collection name from the project outline.
const CollectionTicketInventory = "ticket_inventory"

// CollectionTicketHolds stores active and completed holds keyed by bookingId (_id).
const CollectionTicketHolds = "ticket_holds"

// InventoryRepository persists inventory and hold documents.
type InventoryRepository struct {
	inv   *mongo.Collection
	holds *mongo.Collection
}

// NewInventoryRepository builds repository handles for both collections.
func NewInventoryRepository(db *mongo.Database) *InventoryRepository {
	return &InventoryRepository{
		inv:   db.Collection(CollectionTicketInventory),
		holds: db.Collection(CollectionTicketHolds),
	}
}

// Ping checks connectivity to MongoDB.
func (r *InventoryRepository) Ping(ctx context.Context) error {
	return r.inv.Database().Client().Ping(ctx, nil)
}

// EnsureIndexes creates required indexes (idempotent).
func (r *InventoryRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.inv.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "eventId", Value: 1}, {Key: "ticketType", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("inventory indexes: %w", err)
	}
	_, err = r.holds.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "expiresAt", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("hold indexes: %w", err)
	}
	return nil
}

// InsertInventory inserts a new inventory row.
func (r *InventoryRepository) InsertInventory(ctx context.Context, doc *domain.Inventory) error {
	if doc.InventoryID == "" {
		doc.InventoryID = newID("inv")
	}
	now := time.Now().UTC()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now
	_, err := r.inv.InsertOne(ctx, doc)
	return err
}

// InsertInventoryMany inserts many inventory rows (bulk setup).
func (r *InventoryRepository) InsertInventoryMany(ctx context.Context, docs []interface{}) error {
	if len(docs) == 0 {
		return nil
	}
	_, err := r.inv.InsertMany(ctx, docs)
	return err
}

// FindInventoryByID loads one inventory row by inventoryId (_id).
func (r *InventoryRepository) FindInventoryByID(ctx context.Context, inventoryID string) (*domain.Inventory, error) {
	var out domain.Inventory
	err := r.inv.FindOne(ctx, bson.M{"_id": inventoryID}).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// FindInventoriesByEvent returns all categories for an event.
func (r *InventoryRepository) FindInventoriesByEvent(ctx context.Context, eventID string) ([]domain.Inventory, error) {
	cur, err := r.inv.Find(ctx, bson.M{"eventId": eventID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var list []domain.Inventory
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// FindInventoryByEventAndType finds the inventory row for an event and ticket type.
func (r *InventoryRepository) FindInventoryByEventAndType(ctx context.Context, eventID string, ticketType domain.TicketType) (*domain.Inventory, error) {
	var out domain.Inventory
	err := r.inv.FindOne(ctx, bson.M{"eventId": eventID, "ticketType": ticketType}).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ReplaceInventory persists the full document (caller sets UpdatedAt).
func (r *InventoryRepository) ReplaceInventory(ctx context.Context, doc *domain.Inventory) error {
	doc.UpdatedAt = time.Now().UTC()
	_, err := r.inv.ReplaceOne(ctx, bson.M{"_id": doc.InventoryID}, doc)
	return err
}

// ReserveHeld attempts to move quantity from available to held. Returns the post-update document.
func (r *InventoryRepository) ReserveHeld(ctx context.Context, inventoryID string, qty int) (*domain.Inventory, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("invalid quantity")
	}
	after := options.After
	opts := options.FindOneAndUpdate().SetReturnDocument(after)

	var out domain.Inventory
	err := r.inv.FindOneAndUpdate(ctx,
		bson.M{
			"_id":               inventoryID,
			"availableQuantity": bson.M{"$gte": qty},
		},
		bson.M{
			"$inc": bson.M{"heldQuantity": qty, "availableQuantity": -qty},
			"$set": bson.M{"updatedAt": time.Now().UTC()},
		},
		opts,
	).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RollbackReserve reverses ReserveHeld when a hold record could not be created.
func (r *InventoryRepository) RollbackReserve(ctx context.Context, inventoryID string, qty int) error {
	if qty <= 0 {
		return nil
	}
	_, err := r.inv.UpdateOne(ctx,
		bson.M{"_id": inventoryID},
		bson.M{
			"$inc": bson.M{"heldQuantity": -qty, "availableQuantity": qty},
			"$set": bson.M{"updatedAt": time.Now().UTC()},
		},
	)
	return err
}

// ConfirmHeld converts held stock to sold for an inventory row.
func (r *InventoryRepository) ConfirmHeld(ctx context.Context, inventoryID string, qty int) (*domain.Inventory, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("invalid quantity")
	}
	after := options.After
	opts := options.FindOneAndUpdate().SetReturnDocument(after)

	var out domain.Inventory
	err := r.inv.FindOneAndUpdate(ctx,
		bson.M{
			"_id":             inventoryID,
			"heldQuantity":    bson.M{"$gte": qty},
		},
		bson.M{
			"$inc": bson.M{"heldQuantity": -qty, "soldQuantity": qty},
			"$set": bson.M{"updatedAt": time.Now().UTC()},
		},
		opts,
	).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ReleaseHeld returns quantity from held back to available.
func (r *InventoryRepository) ReleaseHeld(ctx context.Context, inventoryID string, qty int) (*domain.Inventory, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("invalid quantity")
	}
	after := options.After
	opts := options.FindOneAndUpdate().SetReturnDocument(after)

	var out domain.Inventory
	err := r.inv.FindOneAndUpdate(ctx,
		bson.M{
			"_id":             inventoryID,
			"heldQuantity":    bson.M{"$gte": qty},
		},
		bson.M{
			"$inc": bson.M{"heldQuantity": -qty, "availableQuantity": qty},
			"$set": bson.M{"updatedAt": time.Now().UTC()},
		},
		opts,
	).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ReplaceHold upserts a hold document keyed by bookingId (_id).
func (r *InventoryRepository) ReplaceHold(ctx context.Context, h *domain.TicketHold) error {
	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now().UTC()
	}
	opts := options.Replace().SetUpsert(true)
	_, err := r.holds.ReplaceOne(ctx, bson.M{"_id": h.BookingID}, h, opts)
	return err
}

// FindHoldByBookingID loads a hold by booking id.
func (r *InventoryRepository) FindHoldByBookingID(ctx context.Context, bookingID string) (*domain.TicketHold, error) {
	var out domain.TicketHold
	err := r.holds.FindOne(ctx, bson.M{"_id": bookingID}).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CountInventoryByEventAndTypeExcluding counts rows matching event and type, optionally excluding an inventory id.
func (r *InventoryRepository) CountInventoryByEventAndTypeExcluding(ctx context.Context, eventID string, ticketType domain.TicketType, excludeID string) (int64, error) {
	filter := bson.M{"eventId": eventID, "ticketType": ticketType}
	if excludeID != "" {
		filter["_id"] = bson.M{"$ne": excludeID}
	}
	return r.inv.CountDocuments(ctx, filter)
}

func newID(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s_fallback_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b)
}

// GenerateInventoryID returns a new unique inventory id suitable for _id / inventoryId.
func GenerateInventoryID() string {
	return newID("inv")
}
