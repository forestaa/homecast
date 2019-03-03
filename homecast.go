package homecast

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/barnybug/go-cast"
	"github.com/barnybug/go-cast/controllers"
	"github.com/micro/mdns"
)

const (
	googleCastServiceName = "_googlecast._tcp"
	googleHomeModelInfo   = "md=Google Home"
)

// CastDevice is cast-able device contains cast client
type CastDevice struct {
	*mdns.ServiceEntry
	client *cast.Client
}

// Connect connects required services to cast
func (g *CastDevice) Connect(ctx context.Context) error {
	return g.client.Connect(ctx)
}

// Close calls client's close func
func (g *CastDevice) Close() {
	g.client.Close()
}

// Speak speaks given text on cast device
func (g *CastDevice) Speak(ctx context.Context, text, lang string) error {
	url, err := tts(text, lang)
	if err != nil {
		return err
	}
	return g.Play(ctx, url)
}

// Play plays media contents on cast device
func (g *CastDevice) Play(ctx context.Context, url *url.URL) error {
	media, err := g.client.Media(ctx)
	if err != nil {
		return err
	}

	mediaItem := controllers.MediaItem{
		ContentId:   url.String(),
		ContentType: "audio/mp3",
		StreamType:  "BUFFERED",
	}

	log.Printf("[INFO] Load media: content_id=%s", mediaItem.ContentId)
	_, err = media.LoadMedia(ctx, mediaItem, 0, true, nil)

	return err
}

// MediaData contains URL and metadata for media
type MediaData struct {
	URL   *url.URL
	Title string
}

// QueueLoad plays playlist on cast device
// See https://developers.google.com/cast/docs/reference/chrome/chrome.cast.media.QueueLoadRequest
func (g *CastDevice) QueueLoad(ctx context.Context, data []MediaData) error {
	media, err := g.client.Media(ctx)
	if err != nil {
		return err
	}

	items := make([]controllers.MediaItem, len(data))
	for i, d := range data {
		items[i] = controllers.MediaItem{
			ContentId:   d.URL.String(),
			StreamType:  "BUFFERED",
			ContentType: "audio/mp3",
			MetaData: controllers.MediaMetaData{
				MetaDataType: 3,
				Title:        d.Title,
			},
		}
	}

	log.Print("[INFO] Queue load")
	_, err = media.QueueLoad(ctx, items, 0, nil)

	return err
}

func (g *CastDevice) QueueInsert(ctx context.Context, data []MediaData) error {
	media, err := g.client.Media(ctx)
	if err != nil {
		return err
	}

	items := make([]controllers.MediaItem, len(data))
	for i, d := range data {
		items[i] = controllers.MediaItem{
			ContentId:   d.URL.String(),
			StreamType:  "BUFFERED",
			ContentType: "audio/mp3",
			MetaData: controllers.MediaMetaData{
				MetaDataType: 3,
				Title:        d.Title,
			},
		}
	}

	log.Print("[INFO] Queue insert")
	_, err = media.QueueInsert(ctx, items, nil)

	return err
}

// LookupAndConnect retrieves cast-able google home devices
func LookupAndConnect(ctx context.Context) []*CastDevice {
	entriesCh := make(chan *mdns.ServiceEntry, 4)

	results := make([]*CastDevice, 0, 4)
	go func() {
		for entry := range entriesCh {
			log.Printf("[INFO] ServiceEntry detected: [%s:%d]%s", entry.AddrV4, entry.Port, entry.Name)
			for _, field := range entry.InfoFields {
				if strings.HasPrefix(field, googleHomeModelInfo) {
					client := cast.NewClient(entry.AddrV4, entry.Port)
					if err := client.Connect(ctx); err != nil {
						log.Printf("[ERROR] Failed to connect: %s", err)
						continue
					}
					results = append(results, &CastDevice{entry, client})
				}
			}
		}
	}()

	err := mdns.Lookup(googleCastServiceName, entriesCh)
	if err != nil {
		log.Printf("[Error] Failed to lookup devices: %v", err)
	}
	close(entriesCh)

	return results
}

// tts provides text-to-speech sound url.
// NOTE: it seems to be unofficial.
func tts(text, lang string) (*url.URL, error) {
	base := "https://translate.google.com/translate_tts?client=tw-ob&ie=UTF-8&q=%s&tl=%s"
	return url.Parse(fmt.Sprintf(base, url.QueryEscape(text), url.QueryEscape(lang)))
}
