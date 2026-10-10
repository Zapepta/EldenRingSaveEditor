package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	WeaponBaseDivisor = 10000
	CategoryTypeMask  = 0x0FFFFFFF
)

type ItemDatabase struct {
	Weapons     map[uint32]string `json:"weapons"`
	Armors      map[uint32]string `json:"armors"`
	Accessories map[uint32]string `json:"accessories"`
	Goods       map[uint32]string `json:"goods"`
	Gems        map[uint32]string `json:"gems"`
}

func downloadParamCategory(fileURL string, isWeaponCategory bool) (map[uint32]string, error) {
	httpResponse, err := http.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("network error downloading %s: %w", fileURL, err)
	}
	defer httpResponse.Body.Close()

	itemsMap := make(map[uint32]string)
	lineScanner := bufio.NewScanner(httpResponse.Body)

	for lineScanner.Scan() {
		rawLine := strings.TrimSpace(lineScanner.Text())

		if rawLine == "" || strings.HasPrefix(rawLine, "#") || strings.HasPrefix(rawLine, "//") {
			continue
		}

		fields := strings.Fields(rawLine)
		if len(fields) < 2 {
			continue
		}

		rawParamID := fields[0]
		itemName := strings.TrimSpace(rawLine[len(rawParamID):])
		itemName = strings.Trim(itemName, `"'`)

		if isIgnoredItemName(itemName) {
			continue
		}

		parsedID, err := parseNumericID(rawParamID)
		if err != nil {
			continue
		}

		normalizedParamID := parsedID
		if isWeaponCategory {
			normalizedParamID = parsedID - (parsedID % WeaponBaseDivisor)
		} else {
			normalizedParamID = parsedID & CategoryTypeMask
		}

		existingName, alreadyExists := itemsMap[normalizedParamID]
		if !alreadyExists || len(itemName) > len(existingName) {
			itemsMap[normalizedParamID] = itemName
		}
	}

	if err := lineScanner.Err(); err != nil {
		return nil, fmt.Errorf("stream read error: %w", err)
	}

	return itemsMap, nil
}

func parseNumericID(rawID string) (uint32, error) {
	if strings.HasPrefix(rawID, "0x") || strings.HasPrefix(rawID, "0X") {
		num64, err := strconv.ParseUint(rawID[2:], 16, 32)
		return uint32(num64), err
	}
	num64, err := strconv.ParseUint(rawID, 10, 32)
	return uint32(num64), err
}

func isIgnoredItemName(name string) bool {
	return name == "" ||
		name == "[ERROR]" ||
		name == "Dummy" ||
		strings.HasPrefix(name, "?ItemName?") ||
		strings.HasPrefix(name, "[DEBUG]")
}

func main() {
	fmt.Println("=== Downloading items... ===")

	const paramdexBaseURL = "https://raw.githubusercontent.com/soulsmods/DSMapStudio/master/src/StudioCore/Assets/Paramdex/ER/Names"

	weapons, err := downloadParamCategory(paramdexBaseURL+"/EquipParamWeapon.txt", true)
	if err != nil {
		fmt.Printf("Error downloading weapons: %v\n", err)
		return
	}

	armors, err := downloadParamCategory(paramdexBaseURL+"/EquipParamProtector.txt", false)
	if err != nil {
		fmt.Printf("Error downloading armor: %v\n", err)
		return
	}

	accessories, err := downloadParamCategory(paramdexBaseURL+"/EquipParamAccessory.txt", false)
	if err != nil {
		fmt.Printf("Error downloading accessories: %v\n", err)
		return
	}

	goods, err := downloadParamCategory(paramdexBaseURL+"/EquipParamGoods.txt", false)
	if err != nil {
		fmt.Printf("Error downloading goods: %v\n", err)
		return
	}

	gems, err := downloadParamCategory(paramdexBaseURL+"/EquipParamGem.txt", false)
	if err != nil {
		fmt.Printf("Error downloading ashes of war: %v\n", err)
		return
	}

	database := ItemDatabase{
		Weapons:     weapons,
		Armors:      armors,
		Accessories: accessories,
		Goods:       goods,
		Gems:        gems,
	}

	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		return
	}

	jsonBytes, err := json.MarshalIndent(database, "", "  ")
	if err != nil {
		fmt.Printf("JSON serialization error: %v\n", err)
		return
	}

	const outputPath = "data/items.json"
	if err := os.WriteFile(outputPath, jsonBytes, 0644); err != nil {
		fmt.Printf("Error writing file %s: %v\n", outputPath, err)
		return
	}
}
