// 略过具体实现，只是提供结构。
package image

import (
	"os"
	"time"
)

// "github.com/docker/docker/client"

type ImageInterface interface {
	Squash() (string, error)
	Format() string
	LoadSquashedImage() error
	ExportTarArchive(string) error
	Cleanup() error
}

type ImageSpec struct {
	ImageID            string
	FromLayer          string
	TmpDir             string
	Tag                string
	Comment            string
	Image              string
	ImageName          string
	ImageTag           string
	LastCreatedBy      string
	SquashID           string
	OCIFormat          bool
	Date               time.Time
	OldImageId         string
	OldImageDir        string
	NewImageDir        string
	SquashedDir        string
	SquashedTar        string
	OldImageLayers     []string
	LayersToSquash     []string
	LayersToMove       []string
	SizeBefore         int64
	SizeAfter          int64
	OldManifest        ImageManifest //Manifest
	OldImageConfig     ImageConfig
	LayerPathsToSquash []string
	LayerPathsToMove   []string
	DiffIDs            []string
	ChainIDs           []string
}

type ImageManifest struct {
	Config       string                  `json:"Config"`
	RepoTags     []string                `json:"RepoTags"`
	Layers       []string                `json:"Layers"`
	LayerSources map[string]LayerDetails `json:"LayerSources"`
}

type LayerDetails struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

type ImageConfig struct {
	Architecture    string        `json:"architecture"`
	Author          string        `json:"author"`
	Config          ConfigDetails `json:"config"`
	ContainerConfig ConfigDetails `json:"container_config"`
	Container       string        `json:"container"`
	LayerID         string        `json:"layerID"`
	Created         string        `json:"created"`
	DockerVersion   string        `json:"docker_version"`
	History         []HistoryItem `json:"history"`
	OS              string        `json:"os"`
	Rootfs          Rootfs        `json:"rootfs"`
	Parent          string        `json:"parent"`
	ID              string        `json:"id"`
}
type ConfigDetails struct {
	Hostname     string                         `json:"Hostname"`
	Domainname   string                         `json:"Domainname"`
	User         string                         `json:"User"`
	AttachStdin  bool                           `json:"AttachStdin"`
	AttachStdout bool                           `json:"AttachStdout"`
	AttachStderr bool                           `json:"AttachStderr"`
	ExposedPorts map[string]map[string]struct{} `json:"ExposedPorts"`
	Tty          bool                           `json:"Tty"`
	OpenStdin    bool                           `json:"OpenStdin"`
	StdinOnce    bool                           `json:"StdinOnce"`
	Env          []string                       `json:"Env"`
	Cmd          []string                       `json:"Cmd"`
	Healthcheck  Healthcheck                    `json:"Healthcheck"`
	Image        string                         `json:"Image"`
	Volumes      interface{}                    `json:"Volumes"`
	WorkingDir   string                         `json:"WorkingDir"`
	Entrypoint   []string                       `json:"Entrypoint"`
	OnBuild      interface{}                    `json:"OnBuild"`
	Labels       map[string]string              `json:"Labels"`
}

type Healthcheck struct {
	Test []string `json:"Test"`
}

type HistoryItem struct {
	Created    string `json:"created"`
	CreatedBy  string `json:"created_by"`
	Comment    string `json:"comment,omitempty"`
	EmptyLayer bool   `json:"empty_layer,omitempty"`
	Author     string `json:"author,omitempty"`
}

type Rootfs struct {
	Type    string   `json:"type"`
	DiffIds []string `json:"diff_ids"`
}

// Metadata represents the image metadata in a structured format
type Metadata struct {
	OldImageConfig     ImageConfig
	Date               time.Time
	LayersToMove       int
	LayerPathsToMove   int
	LayerPathsToSquash []string
	DiffIDs            []string
	SquashID           string
	Comment            string
}

type FileWithPath struct {
	Path string
	Info os.FileInfo
}
