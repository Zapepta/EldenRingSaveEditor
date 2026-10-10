package main

import (
	"encoding/binary"
	"fmt"
)

const (
	WeaponBaseDivisor     = 10000
	GaItemPoolStartOffset = 0x20
	GaItemPoolCapacity    = 5120
	MaxBackpackCapacity   = 2688
	BackpackEntrySize     = 12
	WeaponStructSize = 21
	ArmorStructSize  = 16
	EmptyStructSize  = 8
	CategoryTypeMask   = 0xF0000000
	CategoryWeaponFlag = 0x80000000
	CategoryArmorFlag  = 0x90000000
	CleanItemIDMask    = 0x0FFFFFFF
	AmmoIDThreshold = 50000000
	PlayerIntermediateBlockSize = 0x1B0 + 0xD0 + 0x58 + 0x1C + 0x58 + 0x58
)

type GaItem struct {
	Offset    int
	Size      int
	Handle    uint32
	ItemID    uint32
	AoWHandle uint32
}

func (item GaItem) Empty() bool      { return item.Handle == 0 || item.Handle == 0xFFFFFFFF }
func (item GaItem) BaseID() uint32   { return item.ItemID - (item.ItemID % WeaponBaseDivisor) }
func (item GaItem) Affinity() uint32 { return (item.ItemID % WeaponBaseDivisor) / 100 }
func (item GaItem) Level() uint32    { return item.ItemID % 100 }
func (item GaItem) CleanID() uint32  { return item.ItemID & CleanItemIDMask }

type BackpackEntry struct {
	Handle    uint32
	Quantity  uint32
	SortIndex uint32
}

type WeaponEntry struct {
	Backpack BackpackEntry
	Item     *GaItem
}

func isRealWeapon(itemInstance *GaItem) bool {
	if itemInstance == nil || itemInstance.Empty() || itemInstance.Handle&CategoryTypeMask != CategoryWeaponFlag {
		return false
	}
	if itemInstance.CleanID() >= AmmoIDThreshold {
		return false
	}
	return true
}

func filterWeapons(backpack []BackpackEntry, itemsByHandle map[uint32]*GaItem) []WeaponEntry {
	weaponsList := make([]WeaponEntry, 0, len(backpack))
	for _, backpackEntry := range backpack {
		if backpackEntry.Handle&CategoryTypeMask != CategoryWeaponFlag {
			continue
		}
		itemInstance := itemsByHandle[backpackEntry.Handle]
		if isRealWeapon(itemInstance) {
			weaponsList = append(weaponsList, WeaponEntry{
				Backpack: backpackEntry,
				Item:     itemInstance,
			})
		}
	}
	return weaponsList
}

func parseGaItems(slotBytes []byte) ([]GaItem, map[uint32]*GaItem, int, error) {
	poolCapacity := GaItemPoolCapacity
	if binary.LittleEndian.Uint32(slotBytes[0:4]) <= 81 {
		poolCapacity = 5118
	}

	itemsList := make([]GaItem, 0, poolCapacity)
	itemsByHandle := make(map[uint32]*GaItem)
	currentCursor := GaItemPoolStartOffset

	for i := 0; i < poolCapacity; i++ {
		if currentCursor+8 > len(slotBytes) {
			break
		}

		itemHandle := binary.LittleEndian.Uint32(slotBytes[currentCursor:])
		paramID := binary.LittleEndian.Uint32(slotBytes[currentCursor+4:])
		entrySize := EmptyStructSize

		switch itemHandle & CategoryTypeMask {
		case CategoryWeaponFlag:
			entrySize = WeaponStructSize
		case CategoryArmorFlag:
			entrySize = ArmorStructSize
		}

		item := GaItem{
			Offset: currentCursor,
			Size:   entrySize,
			Handle: itemHandle,
			ItemID: paramID,
		}

		if itemHandle&CategoryTypeMask == CategoryWeaponFlag && entrySize >= 20 {
			item.AoWHandle = binary.LittleEndian.Uint32(slotBytes[currentCursor+16:])
		}

		itemsList = append(itemsList, item)
		if !item.Empty() {
			itemsByHandle[itemHandle] = &itemsList[len(itemsList)-1]
		}

		currentCursor += entrySize
	}

	return itemsList, itemsByHandle, currentCursor, nil
}

func readBackpackInventory(slotBytes []byte, gaItemEndOffset int) ([]BackpackEntry, error) {
	inventoryHeaderOffset := gaItemEndOffset + PlayerIntermediateBlockSize
	firstItemCursor := inventoryHeaderOffset + 4

	if firstItemCursor+(MaxBackpackCapacity*BackpackEntrySize) > len(slotBytes) {
		return nil, fmt.Errorf("error: backpack array exceeds slot bounds")
	}

	var backpackItems []BackpackEntry
	currentSlotCursor := firstItemCursor

	for i := 0; i < MaxBackpackCapacity; i++ {
		itemHandle := binary.LittleEndian.Uint32(slotBytes[currentSlotCursor:])
		quantity := binary.LittleEndian.Uint32(slotBytes[currentSlotCursor+4:])
		sortIndex := binary.LittleEndian.Uint32(slotBytes[currentSlotCursor+8:])

		currentSlotCursor += BackpackEntrySize

		if itemHandle == 0 || itemHandle == 0xFFFFFFFF {
			continue
		}

		backpackItems = append(backpackItems, BackpackEntry{
			Handle:    itemHandle,
			Quantity:  quantity,
			SortIndex: sortIndex,
		})
	}

	return backpackItems, nil
}

func upgradeWeapon(slotBytes []byte, itemInstance *GaItem, newLevel uint32) {
	newParamID := itemInstance.BaseID() + (itemInstance.Affinity() * 100) + newLevel
	binary.LittleEndian.PutUint32(slotBytes[itemInstance.Offset+4:], newParamID)
	itemInstance.ItemID = newParamID
}
