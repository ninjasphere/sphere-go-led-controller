package ui

import (
	"fmt"
	"image"
	"os"
	"time"

	"github.com/bugsnag/bugsnag-go"
	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/model"
	owm "github.com/ninjasphere/openweathermap"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

var enableWeatherPane = config.MustBool("led.weather.enabled")
var weatherUpdateInterval = config.MustDuration("led.weather.updateInterval")

var globalSite *model.Site
var timezone *time.Location

type WeatherPane struct {
	siteModel  *ninja.ServiceClient
	site       *model.Site
	getWeather *time.Timer
	weather    *owm.CurrentWeatherData
	image      util.Image
}

func NewWeatherPane(conn *ninja.Connection) *WeatherPane {

	pane := &WeatherPane{
		siteModel: conn.GetServiceClient("$home/services/SiteModel"),
		image:     util.LoadImage(util.ResolveImagePath("weather/loading.gif")),
	}

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
}

func (p *WeatherPane) Render() (*image.RGBA, error) {
	return p.image.GetNextFrame(), nil
}

func (p *WeatherPane) IsDirty() bool {
	return true
}
