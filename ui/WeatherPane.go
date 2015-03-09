package ui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"time"

	"github.com/bugsnag/bugsnag-go"
	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/model"
	owm "github.com/ninjasphere/openweathermap"
	"github.com/ninjasphere/sphere-go-led-controller/fonts/O4b03b"
	"github.com/ninjasphere/sphere-go-led-controller/fonts/clock"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

var enableWeatherPane = config.MustBool("led.weather.enabled")
var weatherUpdateInterval = config.MustDuration("led.weather.updateInterval")
var temperatureDisplayTime = config.Duration(time.Second*5, "led.weather.temperatureDisplayTime")

var globalSite *model.Site
var timezone *time.Location

type WeatherPane struct {
	siteModel   *ninja.ServiceClient
	site        *model.Site
	getWeather  *time.Timer
	tempTimeout *time.Timer
	temperature bool
	weather     *owm.ForecastWeatherData
	image       util.Image
}

func NewWeatherPane(conn *ninja.Connection) *WeatherPane {

	pane := &WeatherPane{
		siteModel: conn.GetServiceClient("$home/services/SiteModel"),
		image:     util.LoadImage(util.ResolveImagePath("weather/loading.gif")),
	}

	pane.tempTimeout = time.AfterFunc(0, func() {
		pane.temperature = false
	})

	if !enableWeatherPane {
		return pane
	}

	var err error
	pane.weather, err = owm.NewForecast("C")
	if err != nil {
		log.Warningf("Failed to load weather api:", err)
		enableWeatherPane = false
	} else {
		go pane.GetWeather()
	}

	return pane
}

func (p *WeatherPane) GetWeather() {

	enableWeatherPane = false

	for {
		site := &model.Site{}
		err := p.siteModel.Call("fetch", config.MustString("siteId"), site, time.Second*5)

		if err == nil && (site.Longitude != nil || site.Latitude != nil) {
			p.site = site
			globalSite = site

			if site.TimeZoneID != nil {
				if timezone, err = time.LoadLocation(*site.TimeZoneID); err != nil {
					log.Warningf("error while setting timezone (%s): %s", *site.TimeZoneID, err)
					timezone, _ = time.LoadLocation("Local")
				}
			}
			break
		}

		log.Infof("Failed to get site, or site has no location.")

		time.Sleep(time.Second * 2)
	}

	for {

		p.weather.DailyByCoordinates(
			&owm.Coordinates{
				Longitude: *p.site.Longitude,
				Latitude:  *p.site.Latitude,
			},
			1,
		)

		if len(p.weather.List) > 0 {

			filename := util.ResolveImagePath("weather/" + p.weather.List[0].Weather[0].Icon + ".png")

			if _, err := os.Stat(filename); os.IsNotExist(err) {
				enableWeatherPane = false
				fmt.Printf("Couldn't load image for weather: %s", filename)
				bugsnag.Notify(fmt.Errorf("Unknown weather icon: %s", filename), p.weather)
			} else {
				p.image = util.LoadImage(filename)
				enableWeatherPane = true
			}
		}

		time.Sleep(weatherUpdateInterval)

	}

}

func (p *WeatherPane) IsEnabled() bool {
	return enableWeatherPane && p.weather.Unit != ""
}

func (p *WeatherPane) Gesture(gesture *gestic.GestureMessage) {
	if gesture.Tap.Active() {
		log.Infof("Weather tap!")

		p.temperature = true
		p.tempTimeout.Reset(temperatureDisplayTime)
	}
}

func (p *WeatherPane) Render() (*image.RGBA, error) {
	if p.temperature {
		img := image.NewRGBA(image.Rect(0, 0, 16, 16))

		drawText := func(text string, col color.RGBA, top int) {
			width := clock.Font.DrawString(img, 0, 8, text, color.Black)
			start := int(16 - width - 2)

			//spew.Dump("text", text, "width", width, "start", start)

			O4b03b.Font.DrawString(img, start, top, text, col)
		}

		if p.weather.City.Country == "US" || p.weather.City.Country == "United States of America" {
			drawText(fmt.Sprintf("%dF", int(p.weather.List[0].Temp.Max*(9.0/5)-459.67)), color.RGBA{253, 151, 32, 255}, 1)
			drawText(fmt.Sprintf("%dF", int(p.weather.List[0].Temp.Min*(9.0/5)-459.67)), color.RGBA{69, 175, 249, 255}, 8)
		} else {
			drawText(fmt.Sprintf("%dC", int(p.weather.List[0].Temp.Max-273.15)), color.RGBA{253, 151, 32, 255}, 1)
			drawText(fmt.Sprintf("%dC", int(p.weather.List[0].Temp.Min-273.15)), color.RGBA{69, 175, 249, 255}, 8)
		}

		return img, nil
	} else {
		return p.image.GetNextFrame(), nil
	}
}

func (p *WeatherPane) IsDirty() bool {
	return true
}
