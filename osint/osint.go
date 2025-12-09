package osint

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/TwiN/go-color"
	"github.com/manifoldco/promptui"
)

const (
	authURL      = "https://www.space-track.org/ajaxauth/login"
	queryBaseURL = "https://www.space-track.org/basicspacedata/query"
)

// Login authenticates with Space-Track API using credentials from environment variables.
// Returns an HTTP client with a cookie jar to maintain the session.
func Login() (*http.Client, error) {
	vals := url.Values{}
	vals.Add("identity", os.Getenv("SPACE_TRACK_USERNAME"))
	vals.Add("password", os.Getenv("SPACE_TRACK_PASSWORD"))

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{
		Jar: jar,
	}

	resp, err := client.PostForm(authURL, vals)
	if err != nil {
		return nil, fmt.Errorf("unable to authenticate with Space-Track: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication failed with status code: %d", resp.StatusCode)
	}

	fmt.Println(color.Ize(color.Green, "  [+] Logged in successfully"))
	return client, nil
}

// QuerySpaceTrack sends a GET request to the Space-Track API using the authenticated client.
// Returns the response body as a string.
func QuerySpaceTrack(client *http.Client, endpoint string) (string, error) {
	req, err := http.NewRequest("GET", queryBaseURL+endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create query request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch data from Space-Track: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("query returned non-success status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	return string(body), nil
}

// extractNorad extracts the NORAD ID from a string in the format "Name (NORAD_ID)".
func extractNorad(str string) string {
	start := strings.Index(str, "(")
	end := strings.Index(str, ")")
	if start == -1 || end == -1 || start >= end {
		return ""
	}
	return str[start+1 : end]
}

// PrintNORADInfo fetches and displays TLE data for a satellite identified by its NORAD ID.
func PrintNORADInfo(norad string, name string) {
	client, err := Login()
	if err != nil {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: "+err.Error()))
		return
	}

	endpoint := fmt.Sprintf("/class/gp_history/format/tle/NORAD_CAT_ID/%s/orderby/EPOCH%%20desc/limit/1", norad)
	data, err := QuerySpaceTrack(client, endpoint)
	if err != nil {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: "+err.Error()))
		return
	}

	lines := strings.Split(strings.TrimSpace(data), "\n")

	var lineOne, lineTwo string

	if len(lines) >= 2 {
		lineOne = strings.TrimSpace(lines[0])
		lineTwo = strings.TrimSpace(lines[1])
	} else {
		tleLines := strings.Fields(data)
		if len(tleLines) < 2 {
			fmt.Println(color.Ize(color.Red, "  [!] ERROR: Invalid TLE data - insufficient fields"))
			return
		}

		// Calculate split point, ensuring we can create two meaningful lines
		mid := 9
		if len(tleLines) < 9 {
			mid = len(tleLines) / 2
		}
		if mid > len(tleLines) {
			mid = len(tleLines) / 2
		}
		// Ensure mid is at least 1 to prevent empty lineOne
		if mid < 1 {
			mid = 1
		}
		// Ensure mid is less than len(tleLines) to ensure lineTwo is non-empty
		if mid >= len(tleLines) {
			mid = len(tleLines) - 1
		}

		lineOne = strings.Join(tleLines[:mid], " ")
		lineTwo = strings.Join(tleLines[mid:], " ")
	}

	if !strings.HasPrefix(lineOne, "1 ") {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: Invalid TLE format - line 1 should start with '1 '"))
		return
	}
	if !strings.HasPrefix(lineTwo, "2 ") {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: Invalid TLE format - line 2 should start with '2 '"))
		return
	}

	tle := ConstructTLE(name, lineOne, lineTwo)

	parsingFailed := false

	line1Fields := strings.Fields(lineOne)
	line2Fields := strings.Fields(lineTwo)

	if len(line1Fields) < 4 || len(line2Fields) < 3 {
		parsingFailed = true
	} else if tle.SatelliteCatalogNumber == 0 && tle.InternationalDesignator == "" && tle.ElementSetEpoch == 0.0 {
		parsingFailed = true
	}

	if parsingFailed {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: Failed to parse TLE data"))
		fmt.Println(color.Ize(color.Red, fmt.Sprintf("       Line 1 fields: %d (minimum required: 4)", len(line1Fields))))
		fmt.Println(color.Ize(color.Red, fmt.Sprintf("       Line 2 fields: %d (minimum required: 3)", len(line2Fields))))
		if len(line1Fields) >= 4 && len(line2Fields) >= 3 {
			fmt.Println(color.Ize(color.Red, "       Note: Field count is sufficient, but parsing failed. Check TLE format."))
		}
		return
	}

	PrintTLE(tle)
}

// SelectSatellite fetches a list of satellites from Space-Track and presents them in an interactive menu.
// Returns the selected satellite name with its NORAD ID in parentheses.
func SelectSatellite() string {
	client, err := Login()
	if err != nil {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: "+err.Error()))
		return ""
	}
	endpoint := "/class/satcat/orderby/SATNAME%20asc/limit/10/emptyresult/show"
	data, err := QuerySpaceTrack(client, endpoint)
	if err != nil {
		fmt.Println(color.Ize(color.Red, "  [!] ERROR: "+err.Error()))
		return ""
	}

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

// GenRowString formats a key-value pair into a table row with proper spacing.
func GenRowString(intro string, input string) string {
	var totalCount int = 4 + len(intro) + len(input) + 2
	var useCount = 63 - totalCount
	return "║ " + intro + ": " + input + strings.Repeat(" ", useCount) + " ║"
}

// Option prompts the user for a numeric input within a specified range.
// Returns the selected number, or exits the program if the minimum value is chosen.
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
