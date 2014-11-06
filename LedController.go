package main

import (
	"image"
	"io"
	"log"
	"os"
	"time"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/sphere-go-led-controller/ui"
	"github.com/ninjasphere/sphere-go-led-controller/util"
	"github.com/tarm/goserial"
)

type LedController struct {
	controlEnabled bool
	controlLayout  *ui.PaneLayout
	pairingLayout  *ui.PairingLayout
	conn           *ninja.Connection
	serial         io.ReadWriteCloser
	waiting        chan bool
}

func NewLedController(conn *ninja.Connection) (*LedController, error) {

	c := &serial.Config{Name: "/dev/tty.ledmatrix", Baud: 115200}
	s, err := serial.OpenPort(c)
	if err != nil {
		return nil, err
	}

	// Send a blank image to the led matrix
	util.WriteLEDMatrix(image.NewRGBA(image.Rect(0, 0, 16, 16)), s)

	controller := &LedController{
		conn:          conn,
		pairingLayout: ui.NewPairingLayout(conn),
		serial:        s,
		waiting:       make(chan bool),
	}

	conn.MustExportService(controller, "$node/"+config.Serial()+"/led-controller", &model.ServiceAnnouncement{
		Schema: "/service/led-controller",
	})

	return controller, nil
}

func (c *LedController) start(enableControl bool) {
	c.controlEnabled = enableControl

	frameWritten := make(chan bool)

	go func() {
		for {
			if c.controlEnabled {

				if c.controlLayout == nil {

					log.Println("Enabling layout... clearing LED")

					util.WriteLEDMatrix(image.NewRGBA(image.Rect(0, 0, 16, 16)), c.serial)

					c.controlLayout = getPaneLayout(c.conn)
					log.Println("Finished control layout")
				}

				image, wake, err := c.controlLayout.Render()
				if err != nil {
					log.Fatal(err)
				}

				go func() {
					util.WriteLEDMatrix(image, c.serial)
					frameWritten <- true
				}()

				select {
				case <-frameWritten:
					// All good.
				case <-time.After(10 * time.Second):
					log.Println("Timeout writing to LED matrix. Quitting.")
					os.Exit(1)
					// Timed out writing to the led matrix. For now. Boot!
					//cmd := exec.Command("reboot")
					//output, err := cmd.Output()

					//log.Printf("Output from reboot: %s err: %s", output, err)
				}

				if wake != nil {
					log.Println("Waiting as the UI is asleep")
					select {
					case <-wake:
						log.Println("UI woke up!")
					case <-c.waiting:
						log.Println("Got a command from rpc...")
					}
				}

			} else {

				image, err := c.pairingLayout.Render()
				if err != nil {
					log.Fatal(err)
				}
				util.WriteLEDMatrix(image, c.serial)

			}
		}
	}()
}

func (c *LedController) EnableControl() error {
	c.controlEnabled = true
	c.gotCommand()
	return nil
}

func (c *LedController) DisableControl() error {
	c.controlEnabled = false
	c.gotCommand()
	return nil
}

type PairingCodeRequest struct {
	Code        string `json:"code"`
	DisplayTime int    `json:"displayTime"`
}

func (c *LedController) DisplayPairingCode(req *PairingCodeRequest) error {
	c.controlEnabled = false
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

	c.controlEnabled = false
	c.pairingLayout.ShowColor(col)
	c.gotCommand()
	return nil
}

type IconRequest struct {
	Icon        string `json:"icon"`
	DisplayTime int    `json:"displayTime"`
}

func (c *LedController) DisplayIcon(req *IconRequest) error {
	c.controlEnabled = false
	c.pairingLayout.ShowIcon(req.Icon)
	c.gotCommand()
	return nil
}

func (c *LedController) gotCommand() {
	select {
	case c.waiting <- true:
	default:
	}
}

// Load from a config file instead...
func getPaneLayout(conn *ninja.Connection) *ui.PaneLayout {
	layout, wake := ui.NewPaneLayout(false)

	mediaPane := ui.NewMediaPane(&ui.MediaPaneImages{
		Volume: "images/media-volume-speaker.gif",
		Mute:   "images/media-volume-mute.png",
		Play:   "images/media-play.png",
		Pause:  "images/media-pause.png",
		Stop:   "images/media-stop.png",
		Next:   "images/media-next.png",
	}, conn)
	layout.AddPane(mediaPane)

	if len(os.Getenv("CERTIFICATION")) > 0 {
		layout.AddPane(ui.NewCertPane(conn.GetMqttClient()))
	} else {
		//layout.AddPane(ui.NewTextScrollPane("Exit Music (For A Film)"))

		heaterPane := ui.NewOnOffPane("images/heater-off.png", "images/heater-on.gif", func(state bool) {
			log.Printf("Heater state: %t", state)
		}, conn, "heater")
		layout.AddPane(heaterPane)
	}

	lightPane := ui.NewLightPane("images/light-off.png", "images/light-on.png", func(state bool) {
		log.Printf("Light on-off state: %t", state)
	}, func(state float64) {
		log.Printf("Light color state: %f", state)
	}, conn)
	layout.AddPane(lightPane)

	fanPane := ui.NewOnOffPane("images/fan-off.png", "images/fan-on.gif", func(state bool) {
		log.Printf("Fan state: %t", state)
	}, conn, "fan")

	layout.AddPane(fanPane)

	go func() {
		<-wake
	}()

	go layout.Wake()

	return layout
}
