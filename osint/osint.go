package osint

import (
	"encoding/json"
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/manifoldco/promptui"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	authURL      = "https://www.space-track.org/ajaxauth/login"
	queryBaseURL = "https://www.space-track.org/basicspacedata/query"
)

func Login() []*http.Cookie {
	vals := url.Values{}
	vals.Add("identity", os.Getenv("SPACE_TRACK_USERNAME"))
	vals.Add("password", os.Getenv("SPACE_TRACK_PASSWORD"))

	client := &http.Client{}

	resp, err := client.PostForm(authURL, vals)
	if err != nil {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: Unable to authenticate with Space-Track"))
		fmt.Printf("Error details: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: Authentication failed"))
		fmt.Printf("Status code: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	fmt.Println(color.Ize(color.Green, "  [+] Logged in successfully"))
	return resp.Cookies()
}

func QuerySpaceTrack(cookies []*http.Cookie, endpoint string) string {
	req, err := http.NewRequest("GET", queryBaseURL+endpoint, nil)
	if err != nil {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: Failed to create query request"))
		fmt.Printf("Error details: %v\n", err)
		os.Exit(1)
	}

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: Failed to fetch data from Space-Track"))
		fmt.Printf("Error details: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: Query returned non-success status code"))
		fmt.Printf("Status code: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: Failed to read response body"))
		fmt.Printf("Error details: %v\n", err)
		os.Exit(1)
	}
	return string(body)
}

func extractNorad(str string) string {
	start := strings.Index(str, "(")
	end := strings.Index(str, ")")
	if start == -1 || end == -1 || start >= end {
		return ""
	}
	return str[start+1 : end]
}

func PrintNORADInfo(norad string, name string) {
	//client := &http.Client{}
	cookies := Login()

	endpoint := fmt.Sprintf("/class/gp_history/format/tle/NORAD_CAT_ID/%s/orderby/EPOCH%%20desc/limit/1", norad)
	data := QuerySpaceTrack(cookies, endpoint)

	tleLines := strings.Fields(data)
	if len(tleLines) < 2 {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: Invalid TLE data"))
		return
	}

	mid := (len(tleLines) / 2) + 1
	lineOne := strings.Join(tleLines[:mid], " ")
	lineTwo := strings.Join(tleLines[mid:], " ")
	tle := ConstructTLE(name, lineOne, lineTwo)
	PrintTLE(tle)
}

func SelectSatellite() string {
	//client := &http.Client{}
	cookies := Login()
	endpoint := "/class/satcat/orderby/SATNAME%20asc/limit/10/emptyresult/show"
	data := QuerySpaceTrack(cookies, endpoint)

	var sats []Satellite
	if err := json.Unmarshal([]byte(data), &sats); err != nil {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: Failed to parse satellite data"))
		fmt.Printf("Error details: %v\n", err)
		return ""
	}

	var satStrings []string
	for _, sat := range sats {
		satStrings = append(satStrings, sat.SATNAME+" ("+sat.NORAD_CAT_ID+")")
	}

	prompt := promptui.Select{
		Label: "Select a Satellite 🛰",
		Items: satStrings,
	}
	_, result, err := prompt.Run()
	if err != nil {
		fmt.Println(color.Ize(color.Red, "  [!] PROMPT FAILED"))
		return ""
	}
	return result
}

func GenRowString(intro string, input string) string {
	var totalCount int = 4 + len(intro) + len(input) + 2
	var useCount = 63 - totalCount
	return "║ " + intro + ": " + input + strings.Repeat(" ", useCount) + " ║"
}

func Option(min int, max int) int {
	fmt.Print("\n ENTER INPUT > ")
	var selection string
	fmt.Scanln(&selection)
	num, err := strconv.Atoi(selection)
	if err != nil {
		fmt.Println(color.Ize(color.Red, "  [!] INVALID INPUT"))
		return Option(min, max)
	} else {
		if num == min {
			fmt.Println(color.Ize(color.Blue, " Escaping Orbit..."))
			os.Exit(1)
			return 0
		} else if num > min && num < max+1 {
			return num
		} else {
			fmt.Println(color.Ize(color.Red, "  [!] INVALID INPUT"))
			return Option(min, max)
		}
	}
}
