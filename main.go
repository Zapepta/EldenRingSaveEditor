package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	Reset = "\033[0m"
	Bold  = "\033[1m"
	Green = "\033[38;2;32;120;55m"
	White = "\033[37m"
	Gray  = "\033[90m"
	Red   = "\033[31m"
)

type SaveProfile struct {
	SteamID string
	Path    string
	ModTime time.Time
}

func discoverSaveFiles() []SaveProfile {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return nil
	}

	eldenRingDir := filepath.Join(appData, "EldenRing")
	entries, err := os.ReadDir(eldenRingDir)
	if err != nil {
		return nil
	}

	var foundSaves []SaveProfile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		if len(dirName) == 17 && strings.HasPrefix(dirName, "7656119") {
			for _, saveFileName := range []string{"ER0000.sl2", "ER0000.co2"} {
				candidatePath := filepath.Join(eldenRingDir, dirName, saveFileName)
				info, err := os.Stat(candidatePath)
				if err == nil && !info.IsDir() {
					foundSaves = append(foundSaves, SaveProfile{
						SteamID: dirName + " (" + saveFileName + ")",
						Path:    candidatePath,
						ModTime: info.ModTime(),
					})
				}
			}
		}
	}

	sort.Slice(foundSaves, func(i, j int) bool {
		return foundSaves[i].ModTime.After(foundSaves[j].ModTime)
	})

	return foundSaves
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func inputPrompt(scanner *bufio.Scanner, message string) string {
	fmt.Print(message)
	if !scanner.Scan() {
		return ""
	}
	return strings.TrimSpace(scanner.Text())
}

func isCancelInput(input string) bool {
	return input == "" || input == "0" || strings.ToLower(input) == "q"
}

type SearchResult struct {
	ID   uint32
	Name string
}

func findWeaponsByQuery(searchQuery string) []SearchResult {
	searchQuery = strings.ToLower(searchQuery)
	var foundWeapons []SearchResult

	for weaponID, weaponName := range DB.Weapons {
		if strings.Contains(strings.ToLower(weaponName), searchQuery) {
			foundWeapons = append(foundWeapons, SearchResult{
				ID:   weaponID,
				Name: weaponName,
			})
		}
	}

	sort.Slice(foundWeapons, func(i, j int) bool {
		return foundWeapons[i].Name < foundWeapons[j].Name
	})

	return foundWeapons
}

func printBanner() {
	fmt.Printf("%s%s", Green, Bold)
	fmt.Println(`  ___ _    ___  ___ _  _   ___ ___ _  _  ___ `)
	fmt.Println(` | __| |  |   \| __| \| | | _ \_ _| \| |/ __|`)
	fmt.Println(` | _|| |__| |) | _|| .` + "`" + ` | |   /| || .` + "`" + ` | (_ |`)
	fmt.Println(` |___|____|___/|___|_|\_| |_|_\___|_|\_|\___|`)
	fmt.Printf("          %sSAVE EDITOR %s•%s PC BND4\n\n%s", White, Gray, Green, Reset)
}

func main() {
	clearScreen()
	printBanner()

	scanner := bufio.NewScanner(os.Stdin)

	if err := initItemDB(); err != nil {
		fmt.Printf("%s[✗] Database error:%s %v\n", Red, Reset, err)
		return
	}

	saveFilePath := ""
	if len(os.Args) > 1 {
		saveFilePath = os.Args[1]
	} else {
		discoveredSaves := discoverSaveFiles()
		if len(discoveredSaves) == 0 {
			if _, err := os.Stat("ER0000.sl2"); err == nil {
				saveFilePath = "ER0000.sl2"
			} else {
				fmt.Printf("%s[✗] Error: No ER0000.sl2 found in AppData or current folder.%s\n", Red, Reset)
				return
			}
		} else if len(discoveredSaves) == 1 {
			saveFilePath = discoveredSaves[0].Path
		} else {
			fmt.Printf("%s┌─ SELECT ACCOUNT ──────────────────────────────────────┐%s\n", Green, Reset)
			for i, saveProfile := range discoveredSaves {
				fmt.Printf("│  %s[%d]%s %-18s │ Last modified: %s │\n",
					Green, i+1, Reset, saveProfile.SteamID, saveProfile.ModTime.Format("02.01.2006 15:04"))
			}
			fmt.Printf("%s└─────────────────────────────────────────────────────────────┘%s\n", Green, Reset)

			choice := inputPrompt(scanner, "\nSelect account (Enter to cancel): ")
			if isCancelInput(choice) {
				return
			}
			idx, err := strconv.Atoi(choice)
			if err != nil || idx < 1 || idx > len(discoveredSaves) {
				fmt.Printf("%sCancelled.%s\n", Red, Reset)
				return
			}
			saveFilePath = discoveredSaves[idx-1].Path
		}
	}

	saveFile, err := LoadSaveFile(saveFilePath)
	if err != nil {
		fmt.Printf("%s[✗] Error:%s Failed to open %s (%v)\n", Red, Reset, saveFilePath, err)
		return
	}

	fmt.Printf("%s[✓]%s Loaded %s%s%s %s(backup: %s.bak)%s\n", Green, Reset, Bold, saveFilePath, Reset, Gray, saveFilePath, Reset)

	activeCharacters := saveFile.ScanCharacters()
	if len(activeCharacters) == 0 {
		fmt.Printf("%s[!] No active characters found.%s\n", Red, Reset)
		return
	}

	fmt.Printf("\n%s┌─ SELECT CHARACTER ───────────────────────────────────────────────────────┐%s\n", Green, Reset)
	for i, character := range activeCharacters {
		fmt.Printf("│  %s[%d]%s Slot %d: %-15s │ Level: %s%3d%s │ %-10s │ Runes: %s%8d%s │\n",
			Green, i+1, Reset, character.Slot, character.Name,
			Green, character.Level, Reset, character.ClassName, Green, character.RealRunes, Reset)
	}
	fmt.Printf("%s└──────────────────────────────────────────────────────────────────────────┘%s\n", Green, Reset)

	choiceString := inputPrompt(scanner, fmt.Sprintf("\n%sSelect character %s(Enter to cancel):%s ", White, Gray, Reset))
	if isCancelInput(choiceString) {
		fmt.Printf("%sCancelled.%s\n", Red, Reset)
		return
	}

	selectedCharIndex, err := strconv.Atoi(choiceString)
	if err != nil || selectedCharIndex < 1 || selectedCharIndex > len(activeCharacters) {
		fmt.Printf("%sCancelled.%s\n", Red, Reset)
		return
	}

	selectedChar := activeCharacters[selectedCharIndex-1]
	slotBytes, _ := saveFile.GetSlotData(selectedChar.Slot)

	for {
		clearScreen()
		printBanner()

		fmt.Printf("%s┌───────────────────────────────────────────────────────────┐%s\n", Green, Reset)
		fmt.Printf("│  %sCharacter :%s %-12s │ %sLevel :%s %s%-4d%s │ %sSlot :%s [%d]    │\n",
			Gray, Reset, selectedChar.Name, Gray, Reset, Green, selectedChar.Level, Reset, Gray, Reset, selectedChar.Slot)
		fmt.Printf("│  %sClass     :%s %-12s │ %sRunes :%s %s%-10d%s            │\n",
			Gray, Reset, selectedChar.ClassName, Gray, Reset, Green, selectedChar.RealRunes, Reset)
		fmt.Printf("%s└───────────────────────────────────────────────────────────┘%s\n", Green, Reset)

		fmt.Printf("  %s[1]%s Weapons & Upgrades\n", Green, Reset)
		fmt.Printf("  %s[2]%s Edit Runes\n", Green, Reset)
		fmt.Printf("  %s[3]%s Attributes / Stats\n", Green, Reset)
		fmt.Printf("  %s[4]%s Change Starting Class\n", Green, Reset)
		fmt.Printf("  %s[5]%s Save to Disk (Recalculate MD5)\n", Green, Reset)
		fmt.Printf("  %s[0]%s Exit without Saving\n", Green, Reset)
		fmt.Printf("%s───────────────────────────────────────────────────────────%s\n", Gray, Reset)

		menuChoice := inputPrompt(scanner, fmt.Sprintf("%sYour choice:%s ", White, Reset))
		if isCancelInput(menuChoice) {
			fmt.Printf("\n%sExiting without changes.%s\n", Gray, Reset)
			return
		}

		switch menuChoice {
		case "1":
			handleInventoryMenu(scanner, slotBytes)

		case "2":
			runeString := inputPrompt(scanner, fmt.Sprintf("\nCurrent Runes: %s%d%s. Enter new amount %s(Enter to cancel):%s ",
				Green, selectedChar.RealRunes, Reset, Gray, Reset))
			if isCancelInput(runeString) {
				continue
			}

			newRunes, err := strconv.ParseUint(runeString, 10, 32)
			if err != nil {
				fmt.Printf("%s[-] Invalid runes amount.%s\n", Red, Reset)
				inputPrompt(scanner, "\nPress Enter to continue...")
				continue
			}

			saveFile.UpdateCharacterRunes(&selectedChar, uint32(newRunes))
			fmt.Printf("%s[✓] Runes updated to %d!%s\n", Green, newRunes, Reset)
			inputPrompt(scanner, "\nPress Enter to continue...")

		case "3":
			handleStatsMenu(scanner, saveFile, &selectedChar)

		case "4":
			fmt.Printf("\n%sAvailable Classes:%s\n", Bold, Reset)
			for classByteIndex := byte(0); classByteIndex < 10; classByteIndex++ {
				fmt.Printf("  %s[%2d]%s %s\n", Green, classByteIndex+1, Reset, classIDs[classByteIndex])
			}

			classChoiceString := inputPrompt(scanner, fmt.Sprintf("\nSelect class %s(Enter to cancel):%s ", Gray, Reset))
			if isCancelInput(classChoiceString) {
				continue
			}

			chosenClassIndex, err := strconv.Atoi(classChoiceString)
			if err != nil || chosenClassIndex < 1 || chosenClassIndex > 10 {
				fmt.Printf("%s[-] Invalid choice.%s\n", Red, Reset)
				inputPrompt(scanner, "\nPress Enter to continue...")
				continue
			}

			targetClassByte := byte(chosenClassIndex - 1)
			slotBytes[selectedChar.PlayerDataStart+PlayerClassOffset] = targetClassByte
			selectedChar.ClassByte = targetClassByte
			selectedChar.ClassName = classIDs[targetClassByte]
			fmt.Printf("%s[✓] Class changed to «%s»!%s\n", Green, selectedChar.ClassName, Reset)
			inputPrompt(scanner, "\nPress Enter to continue...")

		case "5":
			saveFile.UpdateSlotChecksum(selectedChar.Slot)
			if err := saveFile.Save(); err != nil {
				fmt.Printf("%s[✗] Error saving file: %v%s\n", Red, err, Reset)
			} else {
				fmt.Printf("\n%s[★] SUCCESS: Changes saved and MD5 recalculated!%s\n", Green, Reset)
				fmt.Printf("%sLaunch Elden Ring and enjoy.%s\n\n", Green, Reset)
			}
			return
		}
	}
}

func handleInventoryMenu(scanner *bufio.Scanner, slotBytes []byte) {
	_, itemsByHandle, gaItemEndOffset, err := parseGaItems(slotBytes)
	if err != nil {
		fmt.Printf("%s[✗] Error parsing GaItems: %v%s\n", Red, err, Reset)
		inputPrompt(scanner, "\nPress Enter to continue...")
		return
	}

	backpackItems, err := readBackpackInventory(slotBytes, gaItemEndOffset)
	if err != nil {
		fmt.Printf("%s[✗] Error reading inventory: %v%s\n", Red, err, Reset)
		inputPrompt(scanner, "\nPress Enter to continue...")
		return
	}

	weaponsList := filterWeapons(backpackItems, itemsByHandle)

	for {
		clearScreen()
		printBanner()

		fmt.Printf("%s┌─ CHARACTER WEAPONS ─────────────────────────────────────────┐%s\n", Green, Reset)
		if len(weaponsList) == 0 {
			fmt.Println("│  (no weapons found)                                         │")
		}
		for i, weaponEntry := range weaponsList {
			itemInstance := weaponEntry.Item
			fmt.Printf("│  %s[%2d]%s %-28s %s+%-2d%s %s(ID: %-8d)%s  │\n",
				Green, i+1, Reset, getWeaponName(itemInstance.BaseID()),
				Green, itemInstance.Level(), Reset, Gray, itemInstance.BaseID(), Reset)
		}
		fmt.Printf("%s└─────────────────────────────────────────────────────────────┘%s\n", Green, Reset)
		fmt.Printf("  %s[1]%s Upgrade Weapon\n", Green, Reset)
		fmt.Printf("  %s[2]%s Swap Weapon\n", Green, Reset)
		fmt.Printf("  %s[0]%s Back to Main Menu\n", Green, Reset)
		fmt.Printf("%s─────────────────────────────────────────────────────────────%s\n", Gray, Reset)

		actionChoice := inputPrompt(scanner, fmt.Sprintf("%sYour choice:%s ", White, Reset))
		if isCancelInput(actionChoice) {
			break
		}

		switch actionChoice {
		case "1":
			numString := inputPrompt(scanner, fmt.Sprintf("Weapon number %s(Enter to cancel):%s ", Gray, Reset))
			if isCancelInput(numString) {
				continue
			}

			chosenIndex, err := strconv.Atoi(numString)
			if err != nil || chosenIndex < 1 || chosenIndex > len(weaponsList) {
				fmt.Printf("%s[-] Invalid weapon number.%s\n", Red, Reset)
				inputPrompt(scanner, "\nPress Enter to continue...")
				continue
			}

			targetWeapon := weaponsList[chosenIndex-1].Item
			levelString := inputPrompt(scanner, fmt.Sprintf("Upgrade level %s(Enter to cancel):%s ", Gray, Reset))
			if isCancelInput(levelString) {
				continue
			}

			targetLevel, err := strconv.Atoi(levelString)
			if err != nil || targetLevel < 0 || targetLevel > 25 {
				fmt.Printf("%s[-] Invalid level.%s\n", Red, Reset)
				inputPrompt(scanner, "\nPress Enter to continue...")
				continue
			}

			upgradeWeapon(slotBytes, targetWeapon, uint32(targetLevel))
			fmt.Printf("%s[✓] Weapon upgraded to +%d!%s\n", Green, targetLevel, Reset)
			inputPrompt(scanner, "\nPress Enter to continue...")

		case "2":
			numString := inputPrompt(scanner, fmt.Sprintf("Weapon number to replace %s(Enter to cancel):%s ", Gray, Reset))
			if isCancelInput(numString) {
				continue
			}

			chosenIndex, err := strconv.Atoi(numString)
			if err != nil || chosenIndex < 1 || chosenIndex > len(weaponsList) {
				fmt.Printf("%s[-] Invalid weapon number.%s\n", Red, Reset)
				inputPrompt(scanner, "\nPress Enter to continue...")
				continue
			}

			targetWeapon := weaponsList[chosenIndex-1].Item

			searchQuery := inputPrompt(scanner, fmt.Sprintf("Search weapon name (for example: moonveil, rivers) %s(Enter to cancel):%s ", Gray, Reset))
			if isCancelInput(searchQuery) {
				continue
			}

			searchResults := findWeaponsByQuery(searchQuery)
			if len(searchResults) == 0 {
				fmt.Printf("%s[-] No weapons found matching query.%s\n", Red, Reset)
				inputPrompt(scanner, "\nPress Enter to continue...")
				continue
			}

			fmt.Printf("\n%sSearch Results:%s\n", Bold, Reset)
			maxDisplayCount := 15
			if len(searchResults) < maxDisplayCount {
				maxDisplayCount = len(searchResults)
			}
			for i := 0; i < maxDisplayCount; i++ {
				fmt.Printf("  %s[%2d]%s %s %s(ID: %d)%s\n",
					Green, i+1, Reset, searchResults[i].Name, Gray, searchResults[i].ID, Reset)
			}

			weaponChoiceString := inputPrompt(scanner, fmt.Sprintf("\nSelect weapon to equip %s(Enter to cancel):%s ", Gray, Reset))
			if isCancelInput(weaponChoiceString) {
				continue
			}

			weaponChoice, err := strconv.Atoi(weaponChoiceString)
			if err != nil || weaponChoice < 1 || weaponChoice > maxDisplayCount {
				continue
			}

			chosenWeapon := searchResults[weaponChoice-1]
			levelString := inputPrompt(scanner, fmt.Sprintf("Desired level %s(Enter to cancel):%s ", Gray, Reset))
			if isCancelInput(levelString) {
				continue
			}

			targetLevel, err := strconv.Atoi(levelString)
			if err != nil || targetLevel < 0 || targetLevel > 25 {
				fmt.Printf("%s[-] Invalid level.%s\n", Red, Reset)
				inputPrompt(scanner, "\nPress Enter to continue...")
				continue
			}

			newParamID := chosenWeapon.ID + uint32(targetLevel)
			binary.LittleEndian.PutUint32(slotBytes[targetWeapon.Offset+4:], newParamID)
			binary.LittleEndian.PutUint32(slotBytes[targetWeapon.Offset+16:], 0xFFFFFFFF)

			targetWeapon.ItemID = newParamID
			targetWeapon.AoWHandle = 0xFFFFFFFF

			fmt.Printf("%s[✓] Successfully equipped: «%s +%d»!%s\n", Green, chosenWeapon.Name, targetLevel, Reset)
			inputPrompt(scanner, "\nPress Enter to continue...")
		}
	}
}

func handleStatsMenu(scanner *bufio.Scanner, saveFile *SaveFile, character *Character) {
	clearScreen()
	printBanner()

	fmt.Printf("%s┌─ CURRENT ATTRIBUTES ────────────────────────────────────────┐%s\n", Green, Reset)
	classMinimums := classBaseStats[character.ClassName]
	for i := 0; i < 8; i++ {
		fmt.Printf("│  %-22s : %s%-2d%s  %s(Class Base: %2d)%s              │\n",
			statNames[i], Green, character.CurrentStats[i], Reset, Gray, classMinimums[i], Reset)
	}
	fmt.Printf("%s└─────────────────────────────────────────────────────────────┘%s\n", Green, Reset)

	fmt.Printf("%sEnter 8 values separated by space %s(Enter to cancel):%s\n> ", White, Gray, Reset)
	userInput := inputPrompt(scanner, "")
	if isCancelInput(userInput) {
		return
	}

	statTokens := strings.Fields(userInput)
	if len(statTokens) != 8 {
		fmt.Printf("%s[-] Error: exactly 8 numbers required!%s\n", Red, Reset)
		inputPrompt(scanner, "\nPress Enter to continue...")
		return
	}

	var newStatsList []uint32
	for statIndex, token := range statTokens {
		statValue, err := strconv.Atoi(token)
		if err != nil || statValue < 1 || statValue > 99 {
			fmt.Printf("%s[-] Stats must be numbers between 1 and 99!%s\n", Red, Reset)
			inputPrompt(scanner, "\nPress Enter to continue...")
			return
		}
		if uint32(statValue) < classMinimums[statIndex] {
			fmt.Printf("%s[-] Warning: %s cannot be lower than class base (%d)!%s\n",
				Red, statNames[statIndex], classMinimums[statIndex], Reset)
			inputPrompt(scanner, "\nPress Enter to continue...")
			return
		}
		newStatsList = append(newStatsList, uint32(statValue))
	}

	saveFile.UpdateCharacterStats(character, newStatsList)
	fmt.Printf("%s[✓] Stats updated! New calculated level: %d%s\n", Green, character.Level, Reset)
	inputPrompt(scanner, "\nPress Enter to continue...")
}
