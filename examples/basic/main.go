package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/samratpro/chrome-manager-go/chromemanager"
)

func main() {
	debugPort := 9221 // 9222, 9223
	manager, err := chromemanager.NewChromeManager("", "", debugPort)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.CloseBrowser()

	// Helper: ask user yes/no
	askUpdate := func(profile string) bool {
		fmt.Printf("Profile '%s' exists. Do you want to update it? (y/n): ", profile)
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			return strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" ||
				strings.ToLower(strings.TrimSpace(scanner.Text())) == "yes"
		}
		return false
	}

	// === Profile 1 ===
	profileName1 := "my_facebook_profile"
	if !manager.ProfileExists(profileName1) {
		fmt.Printf("Profile '%s' does not exist. Setting it up now.\n", profileName1)
		if err := manager.SetupProfile(
			profileName1,
			"https://www.facebook.com",
			"Please login to Facebook, then close the browser to save your session.",
			false,
		); err != nil {
			log.Fatal(err)
		}
	} else if askUpdate(profileName1) {
		if err := manager.SetupProfile(
			profileName1,
			"https://www.facebook.com",
			"Please update your Facebook session, then close the browser to save.",
			false,
		); err != nil {
			log.Fatal(err)
		}
	}

	// === Profile 2 ===
	profileName2 := "my_facebook_profile2"
	if !manager.ProfileExists(profileName2) {
		fmt.Printf("Profile '%s' does not exist. Setting it up now.\n", profileName2)
		if err := manager.SetupProfile(
			profileName2,
			"https://www.facebook.com",
			"Please login to Facebook, then close the browser to save your session.",
			false,
		); err != nil {
			log.Fatal(err)
		}
	} else if askUpdate(profileName2) {
		if err := manager.SetupProfile(
			profileName2,
			"https://www.facebook.com",
			"Please update your Facebook session, then close the browser to save.",
			false,
		); err != nil {
			log.Fatal(err)
		}
	}

	// === Use Profile 1 ===
	{
		page, err := manager.ConnectToBrowser(profileName1, "https://www.facebook.com", false, 60000)
		if err != nil {
			log.Fatal(err)
		}
		title, _ := page.Title()
		fmt.Println("Page Title:", title)
		fmt.Println("Press Enter to close the browser...")
		fmt.Scanln()
		page.Close()
		manager.CloseBrowser()
	}

	// === Use Profile 2 ===
	{
		page, err := manager.ConnectToBrowser(profileName2, "https://www.facebook.com", false, 60000)
		if err != nil {
			log.Fatal(err)
		}
		title, _ := page.Title()
		fmt.Println("Page Title:", title)
		fmt.Println("Press Enter to close the browser...")
		fmt.Scanln()
		page.Close()
		manager.CloseBrowser()
	}
}
