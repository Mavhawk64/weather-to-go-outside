package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// IP Stack API

type IpStack struct {
	Ip             string
	Type           string
	Continent_code string
	Continent_name string
	Country_code   string
	Country_name   string
	Region_code    string
	Region_name    string
	City           string
	Zip            string
	Latitude       float64
	Longitude      float64
	Location       Location
}

type Location struct {
	Geoname_id                 int
	Capital                    string
	Languages                  []Language
	Country_flag               string
	Country_flag_emoji         string
	Country_flag_emoji_unicode string
	Calling_code               string
	Is_eu                      bool
}

type Language struct {
	Code   string
	Name   string
	Native string
}

// Seven Day Forecast API

type SevenDayForecast struct {
	Product    string           `json:"product"`
	Init       string           `json:"init"`
	DataSeries []DataSeries7Day `json:"dataseries"`
}

type DataSeries7Day struct {
	Date        int    `json:"date"`
	Weather     string `json:"weather"`
	Temp2m      MinMax `json:"temp2m"`
	Wind10m_Max int    `json:"wind10m_max"`
}

type MinMax struct {
	Max int `json:"max"`
	Min int `json:"min"`
}

// Civil Forecast API

type CivilForecast struct {
	Product    string
	Init       string
	DataSeries []DataSeriesCivil
}

type DataSeriesCivil struct {
	Timepoint    int
	Cloudcover   int
	Lifted_index int
	Prec_type    string
	Prec_amount  float32
	Temp2m       int
	Rh2m         string
	Wind10m      Wind10m // Note: Direction is NEWS
	Weather      string
}

type Wind10m struct {
	Direction string
	Speed     int
}

// METEO Forecast API

type MeteoForecast struct {
	Product    string
	Init       string
	DataSeries []DataSeriesMeteo
}

type DataSeriesMeteo struct {
	Timepoint    int
	Cloudcover   int
	Highcloud    int
	Midcloud     int
	Lowcloud     int
	Rh_profile   []RhProfile
	Wind_profile []WindProfile
	Temp2m       int
	Lifted_index int
	Rh2m         int
	Msl_pressure int
	Wind10m      Wind10m // Note: Direction is in degrees
	Prec_type    string
	Prec_amount  float32
	Snow_depth   int
}

type RhProfile struct {
	Layer string
	Rh    int
}

type WindProfile struct {
	Layer     string
	Direction int
	Speed     int
}

// Combined struct of CIVIL and METEO forecast data
type WeatherForecast struct {
	Product    string
	Init       string
	DataSeries []DataSeriesWeather
}

type DataSeriesWeather struct {
	Datetime     string
	Cloudcover   int
	Cloudprofile Cloudprofile
	Rh_profile   []RhProfile
	Wind_profile []WindProfile
	Prec_type    string
	Prec_amount  float32
	Temp2m       int
	Rh2m         string
	Lifted_index int
	Msl_pressure int
	Wind10m      Wind10m
	Snow_depth   int
	Weather      string
}

type Cloudprofile struct {
	Highcloud int
	Midcloud  int
	Lowcloud  int
}

type MyTime struct {
	Year   int
	Month  int
	Day    int
	Hour   int
	Minute int
	Second int
}

func time_to_string(t MyTime) string {
	return fmt.Sprintf("%02d/%02d/%04d %02d:%02d:%02d", t.Month, t.Day, t.Year, t.Hour, t.Minute, t.Second)
}

func add_hours(t MyTime, hours int) MyTime {
	t.Hour += hours
	for t.Hour >= 24 {
		t.Hour -= 24
		t.Day += 1
	}

	if t.Day > 31 {
		t.Day -= 31
		t.Month += 1
	}

	if intInSlice(t.Month, []int{4, 6, 9, 11}) && t.Day > 30 {
		t.Day -= 30
		t.Month += 1
	}

	// If February and leap year
	if t.Month == 2 && t.Day > 29 && (t.Year%4 == 0 && (t.Year%100 != 0 || t.Year%400 == 0)) {
		t.Day -= 29
		t.Month += 1
	}
	// If February and not leap year
	if t.Month == 2 && t.Day > 28 && !(t.Year%4 == 0 && (t.Year%100 != 0 || t.Year%400 == 0)) {
		t.Day -= 28
		t.Month += 1
	}

	if t.Month > 12 {
		t.Month -= 12
		t.Year += 1
	}

	return t
}

func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Combine the weather data from both the CIVIL and METEO APIs
func combine_weather_data(civil CivilForecast, meteo MeteoForecast) WeatherForecast {
	var weather WeatherForecast
	weather.Product = "Weather Forecast"
	init_time := string_to_time(civil.Init)
	weather.Init = time_to_string(init_time)
	if len(meteo.DataSeries) != len(civil.DataSeries) {
		log.Fatal("The METEO and CIVIL APIs returned different numbers of data points")
	}
	weather.DataSeries = make([]DataSeriesWeather, len(meteo.DataSeries))
	for i := 0; i < len(meteo.DataSeries); i++ {
		// Take the Hi-Res data from each API (2m RH from CIVIL, 10m wind from METEO)
		var data DataSeriesWeather
		data.Datetime = time_to_string(add_hours(init_time, meteo.DataSeries[i].Timepoint))
		data.Cloudcover = meteo.DataSeries[i].Cloudcover
		data.Cloudprofile.Highcloud = meteo.DataSeries[i].Highcloud
		data.Cloudprofile.Midcloud = meteo.DataSeries[i].Midcloud
		data.Cloudprofile.Lowcloud = meteo.DataSeries[i].Lowcloud
		data.Rh_profile = meteo.DataSeries[i].Rh_profile
		data.Wind_profile = meteo.DataSeries[i].Wind_profile
		data.Prec_type = meteo.DataSeries[i].Prec_type
		data.Prec_amount = meteo.DataSeries[i].Prec_amount
		data.Temp2m = meteo.DataSeries[i].Temp2m
		data.Rh2m = civil.DataSeries[i].Rh2m
		data.Lifted_index = meteo.DataSeries[i].Lifted_index
		data.Msl_pressure = meteo.DataSeries[i].Msl_pressure
		data.Wind10m = meteo.DataSeries[i].Wind10m
		data.Snow_depth = meteo.DataSeries[i].Snow_depth
		data.Weather = civil.DataSeries[i].Weather
		weather.DataSeries[i] = data
	}
	return weather
}

// Map numbers to values (if it is a range, I will use the average)

var windSpeed10m = []string{"Undefined / No Data", "Below 0.3m/s (calm)", ".3-3.4m/s (light)", "3.4-8.0m/s (moderate)", "8.0-10.8m/s (fresh)", "10.8-17.2m/s (strong)", "17.2-24.5m/s (gale)", "24.5-32.6m/s (storm)", "Over 32.6m/s (hurricane)"}

func main() {
	seven_day_forecast_url := "https://www.7timer.info/bin/civillight.php?lon=&lat=&ac=0&unit=metric&output=json&tzshift=0" // replace lon= with lon=### and lat= with lat=### to get weather for a specific location

	// Idea: create civil and meteo structs, and combine them because they both span 192 hours
	civil_forecast_url := "https://www.7timer.info/bin/civil.php?lon=&lat=&ac=0&unit=metric&output=json&tzshift=0" // replace lon= with lon=### and lat= with lat=### to get weather for a specific location
	meteo_forecast_url := "https://www.7timer.info/bin/meteo.php?lon=&lat=&ac=0&unit=metric&output=json&tzshift=0" // replace lon= with lon=### and lat= with lat=### to get weather for a specific location

	// is_imperial := true

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// Commented code reads the current directory and lists all files
	/** files, err := os.ReadDir(".")
	if err != nil {
		fmt.Println(err)
	}
	for _, file := range files {
		fmt.Println(file.Name())
	} **/

	// Get the API key from the environment
	ipstack_key := os.Getenv("IPSTACK_KEY")

	fmt.Println(ipstack_key)

	// respip, err := http.Get("http://api.ipstack.com/check?access_key=" + ipstack_key)
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// // We Read the response body on the line below.
	// bodyip, err := ioutil.ReadAll(respip.Body)
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// DEBUG
	// fmt.Println(string(body))

	bodyip := `{"ip": "174.63.144.133", "type": "ipv4", "continent_code": "NA", "continent_name": "North America", "country_code": "US", "country_name": "United States", "region_code": "FL", "region_name": "Florida", "city": "Estero", "zip": "33928", "latitude": 26.43515968322754, "longitude": -81.8109130859375, "location": {"geoname_id": 4154568, "capital": "Washington D.C.", "languages": [{"code": "en", "name": "English", "native": "English"}], "country_flag": "https://assets.ipstack.com/flags/us.svg", "country_flag_emoji": "\ud83c\uddfa\ud83c\uddf8", "country_flag_emoji_unicode": "U+1F1FA U+1F1F8", "calling_code": "1", "is_eu": false}}`

	var ipstack IpStack
	json.Unmarshal([]byte(bodyip), &ipstack)
	fmt.Println(ipstack)
	fmt.Println(ipstack.Ip)
	fmt.Println(ipstack.Latitude)
	fmt.Println(ipstack.Longitude)

	// Replace the lon= and lat= in the url with the latitude and longitude of the current location
	seven_day_forecast_url = strings.Replace(seven_day_forecast_url, "lon=", "lon="+strconv.FormatFloat(ipstack.Longitude, 'f', 6, 64), 1)
	seven_day_forecast_url = strings.Replace(seven_day_forecast_url, "lat=", "lat="+strconv.FormatFloat(ipstack.Latitude, 'f', 6, 64), 1)

	// DEBUG
	fmt.Println(seven_day_forecast_url)

	// Get the JSON data from the API
	resp7, err := http.Get(seven_day_forecast_url)
	if err != nil {
		log.Fatalln(err)
	}

	// We Read the response body on the line below.
	body7, err := ioutil.ReadAll(resp7.Body)
	if err != nil {
		log.Fatalln(err)
	}

	// DEBUG
	// fmt.Println(string(body7))

	// Unmarshal the JSON data into the struct
	var seven_day_forecast SevenDayForecast
	json.Unmarshal([]byte(body7), &seven_day_forecast)

	// DEBUG
	// fmt.Println(seven_day_forecast)

	// Adjust the CIVIL and METEO urls to fill in lat/lon
	civil_forecast_url = strings.Replace(civil_forecast_url, "lon=", "lon="+strconv.FormatFloat(ipstack.Longitude, 'f', 6, 64), 1)
	civil_forecast_url = strings.Replace(civil_forecast_url, "lat=", "lat="+strconv.FormatFloat(ipstack.Latitude, 'f', 6, 64), 1)

	meteo_forecast_url = strings.Replace(meteo_forecast_url, "lon=", "lon="+strconv.FormatFloat(ipstack.Longitude, 'f', 6, 64), 1)
	meteo_forecast_url = strings.Replace(meteo_forecast_url, "lat=", "lat="+strconv.FormatFloat(ipstack.Latitude, 'f', 6, 64), 1)

	// DEBUG
	fmt.Println(civil_forecast_url)
	fmt.Println(meteo_forecast_url)

	// Get the JSON data from both CIVIL and METEO
	respc, err := http.Get(civil_forecast_url)
	if err != nil {
		log.Fatalln(err)
	}
	bodyc, err := ioutil.ReadAll(respc.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var civil_forecast CivilForecast
	json.Unmarshal([]byte(bodyc), &civil_forecast)

	respm, err := http.Get(meteo_forecast_url)
	if err != nil {
		log.Fatalln(err)
	}
	bodym, err := ioutil.ReadAll(respm.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var meteo_forecast MeteoForecast
	json.Unmarshal([]byte(bodym), &meteo_forecast)

	// DEBUG
	// fmt.Println(civil_forecast)
	// fmt.Println(meteo_forecast)

	// fmt.Println(meteo_forecast.DataSeries[0].Wind_profile)

	// Combine the data from the two sources into a single struct
	var weather_forecast = combine_weather_data(civil_forecast, meteo_forecast)

	// DEBUG
	// fmt.Println(weather_forecast)

	// Marshal the data into JSON
	weather_forecast_json, err := json.Marshal(weather_forecast)
	if err != nil {
		log.Fatalln(err)
	}

	// DEBUG
	fmt.Println(string(weather_forecast_json))

	// Save the JSON data to a file
	err = ioutil.WriteFile("weather_forecast.json", weather_forecast_json, 0644)

}

func parse_datetime(x int) string {
	// 2022111812 -> 11/18/2022 12:00:00
	// 0123456789
	year := strconv.Itoa(x)[0:4]
	month := strconv.Itoa(x)[4:6]
	day := strconv.Itoa(x)[6:8]
	hour := strconv.Itoa(x)[8:10]
	return month + "/" + day + "/" + year + " " + hour + ":00:00"
}

func parse_datetime_s(x string) string {
	// 2022111812 -> 11/18/2022 12:00:00
	// 0123456789
	year := x[0:4]
	month := x[4:6]
	day := x[6:8]
	hour := x[8:10]
	return month + "/" + day + "/" + year + " " + hour + ":00:00"
}

func string_to_time(x string) MyTime {
	// 2022111812 -> 11/18/2022 12:00:00
	// 0123456789
	year, _ := strconv.Atoi(x[0:4])
	month, _ := strconv.Atoi(x[4:6])
	day, _ := strconv.Atoi(x[6:8])
	hour, _ := strconv.Atoi(x[8:10])

	mt := MyTime{Year: year, Month: month, Day: day, Hour: hour, Minute: 0, Second: 0}
	return mt
}

func celsius_to_fahrenheit(x int) int {
	return int(float64(x)*1.8 + 32)
}
