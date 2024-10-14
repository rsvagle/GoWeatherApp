package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

// lipgloss styles
var titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ff41"))
var rainStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#add8e6"))
var sunnyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f9d71c"))
var cloudStyle3 = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#ffffff"))
var cloudStyle2 = lipgloss.NewStyle().Foreground(lipgloss.Color("#fffffff"))

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("There's been an error: %v", err)
		os.Exit(1)
	}
}

// initialize the viewModel
func initialModel() viewModel {
	startingLocation := getLocationData()

	// Get the weather from the location
	var weather Weather = getWeatherForLocation(startingLocation.Lat, startingLocation.Lon)

	// put the full weather response into DailyWeather structs
	var dailyWeatherArray []DailyWeather

	// Iterate through the weather data and display it
	for i := 0; i < len(weather.Daily.Time); i++ {
		dailyWeather := DailyWeather{
			Time:           weather.Daily.Time[i],
			WeatherCode:    weather.Daily.WeatherCode[i],
			TemperatureMax: weather.Daily.TemperatureMax[i],
			TemperatureMin: weather.Daily.TemperatureMin[i],
		}
		dailyWeatherArray = append(dailyWeatherArray, dailyWeather)
	}

	return viewModel{
		location:     startingLocation,
		dailyWeather: dailyWeatherArray,
		weather:      weather,
		input:        "",
		message:      "",
	}
}

// init the bubbletea view
func (m viewModel) Init() tea.Cmd {
	return nil
}

// updates the view on input
func (m viewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			inputLower := strings.ToLower(m.input) // make commands case insensitive
			if inputLower == "quit" {
				return m, tea.Quit
			}
			newModel := m.handleInput(m.input)
			return newModel, nil
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case tea.KeyCtrlC:
			return m, tea.Quit
		default:
			m.input += msg.String()
		}
	}
	return m, nil
}

// view for bubble tea
// switch the view based on the viewMode
func (m viewModel) View() string {

	// Print the location and weather data
	s := "\n"
	s += titleStyle.Render(fmt.Sprintf("Weather for %s, %s", m.location.City, m.location.Region))
	s += "\n"
	s += titleStyle.Render(fmt.Sprintf("Latitude: %s, Longitude: %s", m.location.Lat, m.location.Lon))
	s += "\n\n"

	// TODO: Need to go line by line where each line contains all daily weather items and their part
	width := 25
	dates := m.weather.Daily.Time
	highs := m.weather.Daily.TemperatureMax
	lows := m.weather.Daily.TemperatureMin
	codes := m.weather.Daily.WeatherCode

	s += formatDatesLine(dates, width)
	s += "\n"
	s += formatSpaceLine(len(dates), width)
	s += "\n"
	s += formatVisualWeatherLine(codes, width, 1)
	s += "\n"
	s += formatVisualWeatherLine(codes, width, 2)
	s += "\n"
	s += formatVisualWeatherLine(codes, width, 3)
	s += "\n"
	s += formatSpaceLine(len(dates), width)
	s += "\n"
	s += formatWeatherCodeLine(codes, width)
	s += "\n"
	s += formatHighsLine(highs, width)
	s += "\n"
	s += formatLowsLine(lows, width)
	s += "\n\n"

	// Prompt for more input
	s += "\n"
	s += "Enter a city and state (e.g., Los Angeles, CA) to get weather or type 'quit' to exit: \n"
	s += fmt.Sprintf("%s", m.input)

	return s
}

func (m viewModel) handleInput(input string) viewModel {
	// Get location from the input

	// Split the input into city and state
	parts := strings.Split(input, ",")
	if len(parts) < 2 {
		m.message = "Please enter both a city and a state (e.g., Los Angeles, CA)"
	}

	city := strings.TrimSpace(parts[0])
	state := strings.TrimSpace(parts[1])

	// Get the latitude and longitude from the city and state
	lat, lon := getLatLonFromCityState(city, state)

	// Now get weather data for lat/lon
	location := Location{
		City:   city,
		Region: state,
		Lat:    lat,
		Lon:    lon,
	}

	// If we get back 0,0 for lat and lon just put unknown
	if lat == "0.0" && lon == "0.0" {
		location.City = "Unknown"
		location.Region = "Unknown"
	}

	// Get the weather from the location
	var weather Weather = getWeatherForLocation(location.Lat, location.Lon)

	// put the full weather response into DailyWeather structs
	var dailyWeatherArray []DailyWeather

	// Iterate through the weather data and display it
	for i := 0; i < len(weather.Daily.Time); i++ {
		dailyWeather := DailyWeather{
			Time:           weather.Daily.Time[i],
			WeatherCode:    weather.Daily.WeatherCode[i],
			TemperatureMax: weather.Daily.TemperatureMax[i],
			TemperatureMin: weather.Daily.TemperatureMin[i],
		}
		dailyWeatherArray = append(dailyWeatherArray, dailyWeather)
	}

	// Update the model
	m.location = location
	m.dailyWeather = dailyWeatherArray
	m.weather = weather
	m.input = ""

	return m
}

// Try to get the current location data
// Return data for Minneapolis, MN if unable to get current location
func getLocationData() Location {
	// Default location should be Minneapolis
	defaultLocation := Location{
		IP:      "0.0.0.0",
		City:    "Minneapolis",
		Region:  "Minnesota",
		Country: "US",
		LatLon:  "44.98,-93.2638",
		Lat:     "44.98",
		Lon:     "-93.2638",
	}

	// Location data based on the IP address if we can get it
	var ipBasedLocation Location

	// Make a request to ipinfo.io to get current location if possible
	resp, err := http.Get("https://ipinfo.io/json")
	if err != nil {
		fmt.Println("Error:", err)
		return defaultLocation // Return default location on error
	}
	defer resp.Body.Close()

	// Read the body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return defaultLocation // Return default location on error
	}

	err = json.Unmarshal(body, &ipBasedLocation)
	if err != nil {
		fmt.Println("Error:", err)
		return defaultLocation // Return default location on error
	}

	// need to parse out the lat/lon individually for api request
	latLon := strings.SplitN(ipBasedLocation.LatLon, ",", 2)

	ipBasedLocation.Lat = latLon[0]
	ipBasedLocation.Lon = latLon[1]

	return ipBasedLocation
}

func getWeatherForLocation(lat string, lon string) Weather {
	dailyMetrics := "weather_code,temperature_2m_max,temperature_2m_min"
	units := "fahrenheit"

	baseURL := "https://api.open-meteo.com/v1/forecast"

	// build the query string
	queryString := fmt.Sprintf("?latitude=%s&longitude=%s&daily=%s&temperature_unit=%s",
		lat,
		lon,
		dailyMetrics,
		units,
	)

	// construct the full string
	fullURL := baseURL + queryString

	// make the http request for weather data
	res, err := http.Get(fullURL)
	if err != nil {
		fmt.Println("Error:", err)
	}
	defer res.Body.Close()

	// read the body in
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// put the json response into my object
	var weather Weather
	err = json.Unmarshal(body, &weather)
	if err != nil {
		fmt.Println("Error:", err)
	}

	return weather
}

func getLatLonFromCityState(city, state string) (string, string) {
	// Placeholder default location in case of failure
	defaultLat := "0.0"
	defaultLon := "0.0"

	// LocationIQ API key and endpoint
	apiKey := "pk.cd63b67671438fd13619f5b4afadcb8c"
	url := fmt.Sprintf("https://us1.locationiq.com/v1/search.php?key=%s&q=%s,%s&format=json", apiKey, city, state)

	// Make the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error making request to LocationIQ:", err)
		return defaultLat, defaultLon
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return defaultLat, defaultLon
	}

	// Parse the JSON response
	var locationData []LocationIQResponse
	err = json.Unmarshal(body, &locationData)
	if err != nil {
		fmt.Println("Error parsing JSON response:", err)
		return defaultLat, defaultLon
	}

	// Check if any results were returned
	if len(locationData) > 0 {
		lat := locationData[0].Lat
		lon := locationData[0].Lon
		return lat, lon
	}

	// If no data, return default lat/lon
	return defaultLat, defaultLon
}

// Function to get weather description based on code
func getWeatherDescriptionFromCode(code int) string {
	switch code {
	case 0:
		return "Sunny"
	case 1, 2, 3:
		return "Cloudy"
	case 45, 48:
		return "Fog"
	case 51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82:
		return "Rain"
	case 71, 73, 75, 77, 85, 86:
		return "Snow"
	case 95, 96, 99:
		return "Thunderstorm"
	default:
		return "Unknown weather code"
	}
}

func getASCIILine1ForWeather(code int) (string, int) {
	switch code {
	case 0:
		return sunnyStyle.Render("\\ | /"), len("\\ | /")
	case 1, 2, 3:
		return cloudStyle2.Render("  ____"), len("    __")
	case 45, 48:
		return "o o o", len("o o o")
	case 51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82:
		return rainStyle.Render("/ / /"), len("/ / /")
	case 71, 73, 75, 77, 85, 86:
		return "* * * *", len("* * * *")
	case 95, 96, 99:
		return "(   ( )", len("(   ( )")
	default:
		return "Unknown weather code", 1
	}
}

func getASCIILine2ForWeather(code int) (string, int) {
	switch code {
	case 0:
		return sunnyStyle.Render("-- O --"), len("-- O --")
	case 1, 2, 3:
		return cloudStyle2.Render("_(    )"), len("   (  )")
	case 45, 48:
		return "o o o o", len("o o o o")
	case 51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82:
		return rainStyle.Render("/ / / /"), len("/ / / /")
	case 71, 73, 75, 77, 85, 86:
		return " * * *", len(" * * *")
	case 95, 96, 99:
		return "(   (   )", len("(   (   )")
	default:
		return "Unknown weather code", 1
	}
}

func getASCIILine3ForWeather(code int) (string, int) {
	switch code {
	case 0:
		return sunnyStyle.Render("/ | \\"), len("/ | \\")
	case 1, 2, 3:
		return "(____)___)", len("(____)___)")
	case 45, 48:
		return "o o o", len("o o o")
	case 51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82:
		return rainStyle.Render("/ /  /"), len("/ /  /")
	case 71, 73, 75, 77, 85, 86:
		return "* * * *", len("* * * *")
	case 95, 96, 99:
		return "/ / / /", len("/ / / /")
	default:
		return "Unknown weather code", 1
	}
}

func formatDate(dateStr string) string {
	// Parse the input string as a date
	layout := "2006-01-02"
	date, err := time.Parse(layout, dateStr)
	if err != nil {
		return ""
	}

	// Format the date like "Sunday October 13"
	formattedDate := date.Format("Monday January 2")
	return formattedDate
}

// Helper function to format each date chunk to have a fixed width
func formatDatesChunk(text string, width int) string {
	date := formatDate(text)

	// Calculate the padding needed to center the text
	padding := (width - len(date)) / 2
	return fmt.Sprintf("%s%s%s", strings.Repeat(" ", padding), date, strings.Repeat(" ", width-len(date)-padding))
}

// Create a formatted line of text with equal width chunks
func formatDatesLine(dates []string, width int) string {
	chunks := make([]string, len(dates))
	for i, date := range dates {
		chunks[i] = formatDatesChunk(date, width)
	}
	return strings.Join(chunks, " | ")
}

// Helper function to format each high temp chunk to have a fixed width
func formatHighsChunk(high float64, width int) string {
	// convert to string and add
	text := fmt.Sprintf("High %.0f", high)

	// Calculate the padding needed to center the text
	padding := (width - len(text)) / 2
	return fmt.Sprintf("%s%s%s", strings.Repeat(" ", padding), text, strings.Repeat(" ", width-len(text)-padding))
}

// Create a formatted line of text with equal width chunks
func formatHighsLine(highs []float64, width int) string {
	chunks := make([]string, len(highs))
	for i, high := range highs {
		chunks[i] = formatHighsChunk(high, width)
	}
	return strings.Join(chunks, " | ")
}

// Helper function to format each high temp chunk to have a fixed width
func formatLowsChunk(low float64, width int) string {
	// convert to string and add
	text := fmt.Sprintf("Low %.0f", low)

	// Calculate the padding needed to center the text
	padding := (width - len(text)) / 2
	return fmt.Sprintf("%s%s%s", strings.Repeat(" ", padding), text, strings.Repeat(" ", width-len(text)-padding))
}

// Create a formatted line of text with equal width chunks
func formatLowsLine(lows []float64, width int) string {
	chunks := make([]string, len(lows))
	for i, low := range lows {
		chunks[i] = formatLowsChunk(low, width)
	}
	return strings.Join(chunks, " | ")
}

// Create a formatted line of space with equal width chunks
func formatSpaceLine(numOfChunks int, width int) string {
	chunks := make([]string, numOfChunks)
	for i := range chunks {
		chunks[i] = strings.Repeat(" ", width)
	}
	return strings.Join(chunks, " | ")
}

// Helper function to format each weather code chunk to have a fixed width
func formatWeatherCodeChunk(code int, width int) string {
	weatherCode := getWeatherDescriptionFromCode(code)

	// Calculate the padding needed to center the text
	padding := (width - len(weatherCode)) / 2
	return fmt.Sprintf("%s%s%s", strings.Repeat(" ", padding), weatherCode, strings.Repeat(" ", width-len(weatherCode)-padding))
}

// Create a formatted line of text with equal width chunks
func formatWeatherCodeLine(codes []int, width int) string {
	chunks := make([]string, len(codes))
	for i, code := range codes {
		chunks[i] = formatWeatherCodeChunk(code, width)
	}
	return strings.Join(chunks, " | ")
}

// Create a formatted line of text with equal width chunks
func formatVisualWeatherLine(codes []int, width int, lineNumber int) string {
	chunks := make([]string, len(codes))
	for i, code := range codes {
		chunks[i] = formatASCIICodeChunk(code, width, lineNumber)
	}
	return strings.Join(chunks, " | ")
}

// Helper function to format each date chunk to have a fixed width
func formatASCIICodeChunk(code int, width int, lineNumber int) string {
	var weatherASCII string
	var weatherASCIIWidth int

	switch lineNumber {
	case 1:
		weatherASCII, weatherASCIIWidth = getASCIILine1ForWeather(code)
	case 2:
		weatherASCII, weatherASCIIWidth = getASCIILine2ForWeather(code)
	case 3:
		weatherASCII, weatherASCIIWidth = getASCIILine3ForWeather(code)
	default:
		weatherASCII, weatherASCIIWidth = "Unkown", 9
	}

	// Calculate the padding needed to center the text
	padding := (width - weatherASCIIWidth) / 2

	return strings.Repeat(" ", padding) + weatherASCII + strings.Repeat(" ", width-weatherASCIIWidth-padding)
}

// struct to match the API json response
type Location struct {
	IP      string `json:"ip"`
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
	LatLon  string `json:"loc"`

	// Separate lat and lon after getting them from the API req
	Lat string
	Lon string
}

// API response for LocationIQ
type LocationIQResponse struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

// struct to match the API json response
type Weather struct {
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
	Daily     struct {
		WeatherCode    []int     `json:"weather_code"`
		Time           []string  `json:"time"`
		TemperatureMax []float64 `json:"temperature_2m_max"`
		TemperatureMin []float64 `json:"temperature_2m_min"`
	} `json:"daily"`
}

type DailyWeather struct {
	WeatherCode    int
	Time           string
	TemperatureMax float64
	TemperatureMin float64
}

// struct for the view
type viewModel struct {
	input        string
	message      string
	location     Location
	dailyWeather []DailyWeather
	weather      Weather
}
