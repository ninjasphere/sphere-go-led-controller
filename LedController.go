package main

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"net"
	"os"
	"time"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	ledmodel "github.com/ninjasphere/sphere-go-led-controller/model"
	"github.com/ninjasphere/sphere-go-led-controller/remote"
	"github.com/ninjasphere/sphere-go-led-controller/ui"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

var log = logger.GetLogger("sphere-go-led-controller")

var enableRemotePanes = config.Bool(false, "led.remote.enable")
var remotePort = config.Int(3115, "led.remote.port")

var fps Tick = Tick{
	name: "Pane FPS",
}

type LedController struct {
	controlEnabled   bool
	controlRequested bool
	controlRendering bool
	commandReceived  bool

	controlLayout *ui.PaneLayout
	pairingLayout *ui.PairingLayout
	conn          *ninja.Connection
	serial        io.ReadWriteCloser
	waiting       chan bool
}

func NewLedController(conn *ninja.Connection) (*LedController, error) {

	s, err := util.GetLEDConnection()

	if err != nil {
		log.Fatalf("Failed to get connection to LED matrix: %s", err)
	}

	// Send a blank image to the led matrix
	util.WriteLEDMatrix(image.NewRGBA(image.Rect(0, 0, 16, 16)), s)

	controller := &LedController{
		conn:          conn,
		pairingLayout: ui.NewPairingLayout(),
		serial:        s,
		waiting:       make(chan bool),
	}

	conn.MustExportService(controller, "$node/"+config.Serial()+"/led-controller", &model.ServiceAnnouncement{
		Schema: "/service/led-controller",
	})

	conn.MustExportService(controller, "$home/led-controller", &model.ServiceAnnouncement{
		Schema: "/service/led-controller",
	})

	if config.HasString("siteId") {
		log.Infof("Have a siteId, checking if homecloud is running")
		// If we have just started, and we have a site, and homecloud is running... enable control!
		go func() {
			siteModel := conn.GetServiceClient("$home/services/SiteModel")
			for {

				if controller.commandReceived {
					log.Infof("Command has been received, stopping search for homecloud.")
					break
				}

				err := siteModel.Call("fetch", config.MustString("siteId"), nil, time.Second*5)

				if err != nil {
					log.Infof("Fetched site to enableControl. Got err: %s", err)
				} else if err == nil && !controller.commandReceived {
					controller.EnableControl()
					break
				}
				time.Sleep(time.Second * 5)
			}
		}()
	}

	return controller, nil
}

func (c *LedController) start(enableControl bool) {
	c.controlRequested = enableControl

	frameWritten := make(chan bool)

	go func() {
		fps.start()

		for {
			fps.tick()

			if c.controlEnabled {
				// Good to go

				image, wake, err := c.controlLayout.Render()
				if err != nil {
					log.Fatalf("Unable to render()", err)
				}

				go func() {
					util.WriteLEDMatrix(image, c.serial)
					frameWritten <- true
				}()

				select {
				case <-frameWritten:
					// All good.
				case <-time.After(10 * time.Second):
					log.Infof("Timeout writing to LED matrix. Quitting.")
					os.Exit(1)
				}

				if wake != nil {
					log.Infof("Waiting as the UI is asleep")
					select {
					case <-wake:
						log.Infof("UI woke up!")
					case <-c.waiting:
						log.Infof("Got a command from rpc...")
					}
				}

			} else if c.controlRequested && !c.controlRendering {

				// We want to display controls, so lets render the pane

				c.controlRendering = true
				go func() {

					log.Infof("Starting control layout")
					c.controlLayout = getPaneLayout(c.conn)
					c.controlRendering = false
					c.controlEnabled = true
					log.Infof("Finished control layout")

				}()
			}

			if c.controlRendering || !c.controlEnabled {
				// We're either already controlling, or waiting for the pane to render

				image, err := c.pairingLayout.Render()
				if err != nil {
					log.Fatalf("Unable to render()", err)
				}
				util.WriteLEDMatrix(image, c.serial)

			}
		}

	}()
}

func (c *LedController) EnableControl() error {
	log.Infof("Enabling control. Already enabled? %t", c.controlEnabled)
	if !c.controlEnabled {
		if c.controlLayout != nil {
			// Pane layout has already been rendered. Just re-enable control.
			c.controlEnabled = true
		} else {
			c.controlRequested = true
		}
		c.gotCommand()
	}
	return nil
}

func (c *LedController) DisableControl() error {
	log.Infof("Disabling control. Currently enabled? %t", c.controlEnabled)

	c.DisplayIcon(&ledmodel.IconRequest{
		Icon: "spinner-red.gif",
	})

	c.controlEnabled = false
	c.controlRequested = false
	c.gotCommand()
	return nil
}

type PairingCodeRequest struct {
	Code        string `json:"code"`
	DisplayTime int    `json:"displayTime"`
}

func (c *LedController) DisplayPairingCode(req *PairingCodeRequest) error {
	c.pairingLayout.ShowCode(req.Code)
	c.gotCommand()
	return nil
}

type ColorRequest struct {
	Color       string `json:"color"`
	DisplayTime int    `json:"displayTime"`
}

func (c *LedController) DisplayColor(req *ColorRequest) error {
	col, err := colorful.Hex(req.Color)

	if err != nil {
		return err
	}

	c.pairingLayout.ShowColor(col)
	c.gotCommand()
	return nil
}

func (c *LedController) DisplayIcon(req *ledmodel.IconRequest) error {
	log.Infof("Displaying icon: %v", req)
	c.pairingLayout.ShowIcon(req.Icon)
	c.gotCommand()
	return nil
}

func (c *LedController) DisplayDrawing() error {
	c.pairingLayout.ShowDrawing()
	return nil
}

func (c *LedController) Draw(updates *[][]uint8) error {
	c.pairingLayout.Draw(updates)
	return nil
}

func (c *LedController) DisplayResetMode(m *ledmodel.ResetMode) error {
	c.DisableControl()
	fade := m.Duration > 0 && !m.Hold
	loading := false
	var col color.Color
	switch m.Mode {
	case "abort":
		col, _ = colorful.Hex("#FFFFFF")
	case "halt":
		col, _ = colorful.Hex("#CDC9C9")
	case "reboot":
		col, _ = colorful.Hex("#00FF00")
	case "reset-userdata":
		col, _ = colorful.Hex("#FFFF00")
	case "reset-root":
		col, _ = colorful.Hex("#FF0000")
	case "none":
		c.EnableControl()
		return nil
	default:
		loading = true
	}

	if loading {
		c.pairingLayout.ShowIcon("spinner-pink.gif")
	} else if fade {
		c.pairingLayout.ShowFadingShrinkingColor(col, m.Duration)
	} else {
		c.pairingLayout.ShowColor(col)
	}

	c.gotCommand()
	return nil
}

func (c *LedController) DisplayUpdateProgress(p *ledmodel.DisplayUpdateProgress) error {
	c.pairingLayout.ShowUpdateProgress(p.Progress)

	return nil
}

func (c *LedController) gotCommand() {
	select {
	case c.waiting <- true:
	default:
	}
	c.commandReceived = true
}

// Load from a config file instead...
func getPaneLayout(conn *ninja.Connection) *ui.PaneLayout {
	layout, wake := ui.NewPaneLayout(false, conn)

	layout.AddPane(ui.NewClockPane())
	layout.AddPane(ui.NewWeatherPane(conn))
	layout.AddPane(ui.NewGesturePane())
	layout.AddPane(ui.NewGameOfLifePane())
	layout.AddPane(ui.NewMediaPane(conn))
	layout.AddPane(ui.NewCertPane(conn.GetMqttClient()))

	//layout.AddPane(ui.NewTextScrollPane("Exit Music (For A Film)"))
	lampPane := ui.NewOnOffPane(util.ResolveImagePath("lamp2-off.gif"), util.ResolveImagePath("lamp2-on.gif"), func(state bool) {
		log.Debugf("Lamp state: %t", state)
	}, conn, "lamp")
	layout.AddPane(lampPane)

	heaterPane := ui.NewOnOffPane(util.ResolveImagePath("heater-off.png"), util.ResolveImagePath("heater-on.gif"), func(state bool) {
		log.Debugf("Heater state: %t", state)
	}, conn, "heater")
	layout.AddPane(heaterPane)

	brightnessPane := ui.NewLightPane(false, util.ResolveImagePath("light-off.png"), util.ResolveImagePath("light-on.png"), conn)
	layout.AddPane(brightnessPane)

	colorPane := ui.NewLightPane(true, util.ResolveImagePath("light-off.png"), util.ResolveImagePath("light-on.png"), conn)
	layout.AddPane(colorPane)

	fanPane := ui.NewOnOffPane(util.ResolveImagePath("fan-off.png"), util.ResolveImagePath("fan-on.gif"), func(state bool) {
		log.Debugf("Fan state: %t", state)
	}, conn, "fan")

	layout.AddPane(fanPane)

	if enableRemotePanes {
		if err := listenForRemotePanes(layout); err != nil {
			log.Fatalf("Failed to start listening for remote panes: %s", err)
		}
	}

	go func() {
		<-wake
	}()

	go layout.Wake()

	return layout
}

func listenForRemotePanes(layout *ui.PaneLayout) error {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", remotePort))
	if err != nil {
		return err
	}
	log.Infof("Listening for remote panes on :%d", remotePort)
	go func() {
		defer listener.Close()

		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatalf("Error accepting remote pane: %s", err)
			}
			go func() {
				log.Infof("Remote pane connected.")

				pane := remote.NewPane(conn)
				layout.AddPane(pane)
				<-pane.Disconnected
				log.Infof("Remote pane disconnected.")
				layout.RemovePane(pane)
			}()
		}
	}()

	return nil
}

type Tick struct {
	count int
	name  string
}

func (t *Tick) tick() {
	t.count++
}

func (t *Tick) start() {
	go func() {
		for {
			time.Sleep(time.Second)
			//log.Debugf("%s - %d", t.name, t.count)
			t.count = 0
		}
	}()
}
