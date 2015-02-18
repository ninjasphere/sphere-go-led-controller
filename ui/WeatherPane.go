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
	weather     *owm.CurrentWeatherData
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
	pane.weather, err = owm.NewCurrent("C")
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

		p.weather.CurrentByCoordinates(
			&owm.Coordinates{
				Longitude: *p.site.Longitude,
				Latitude:  *p.site.Latitude,
			},
		)

		if len(p.weather.Weather) > 0 {

			filename := util.ResolveImagePath("weather/" + p.weather.Weather[0].Icon + ".png")

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

		var temp string
		if p.weather.Sys.Country == "US" {
			temp = fmt.Sprintf("%dF", int(p.weather.Main.Temp*(9/5)-459.67))
		} else {
			temp = fmt.Sprintf("%dC", int(p.weather.Main.Temp-273.15))
		}

		width := clock.Font.DrawString(img, 0, 0, temp, color.Black)
		start := int((16 - width) / 2)

		O4b03b.Font.DrawString(img, start, 5, temp, color.RGBA{255, 255, 255, 255})

		return img, nil
	} else {
		return p.image.GetNextFrame(), nil
	}
}

func (p *WeatherPane) IsDirty() bool {
	return true
}
