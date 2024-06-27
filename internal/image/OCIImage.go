package image

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type V2Image struct {
	ImageSpec                   // Embedding V1Image to reuse fields
	DockerClient *client.Client // Placeholder for Docker client
	Logger       *logrus.Logger
}

// NewImage creates a new instance of Image with provided parameters.
func NewV2Image(s *Squash) *V2Image {
	return &V2Image{
		ImageSpec: ImageSpec{
			Image:     s.image,
			Tag:       s.tag,
			FromLayer: s.fromLayer,
			TmpDir:    s.tmpDir,

			Comment:       s.comment,
			Date:          time.Now(),
			LastCreatedBy: s.lastCreatedBy,
		},
		DockerClient: s.docker,
		Logger:       s.logs,
	}
}

func (v2 *V2Image) Format() string {
	return "V2"
}

func (v2 *V2Image) Squash() (string, error) {
	// Implementation for V2
	if err := v2.beforeSquashing(); err != nil {
		return "", err
	}
	ret, err := v2.squash()
	if err != nil {
		return "", err
	}
	if err := v2.afterSquashing(); err != nil {
		return "", err
	}
	return ret, nil
}

func (im *V2Image) afterSquashing() error {

	var err error
	im.Logger.Info("Removing from disk already squashed layers...")
	im.Logger.Infof("Cleaning up %s temporary directory...", im.OldImageDir)
	if err = os.RemoveAll(im.OldImageDir); err != nil {
		im.Logger.Errorf("Cleaning up  temporary directory failed:", err)
	}
	im.SizeAfter, err = im.dirSize(im.NewImageDir)
	if err != nil {
		return err
	}

	sizeBeforeMb := float64(im.SizeBefore) / 1024 / 1024
	sizeAfterMb := float64(im.SizeAfter) / 1024 / 1024
	im.Logger.Infof("Original image size: %f MB , Squashed image size: %f MB", sizeBeforeMb, sizeAfterMb)
	if sizeAfterMb > sizeBeforeMb {
		im.Logger.Info("If the squashed image is larger than original it means that there were no meaningful files to squash and it just added metadata. Are you sure you specified correct parameters?")
	} else {

		fmt.Printf("Image size decreased by [ %.2f%% ]\n", float64(((sizeBeforeMb-sizeAfterMb)/sizeBeforeMb)*100))
	}

	return nil
}

func (im *V2Image) squash() (string, error) {

	if len(im.LayerPathsToSquash) != 0 {
		os.Mkdir(im.SquashedDir, os.ModePerm)
		im.squashLayers()
	}

	var layerPathID string
	var oldLayerPath string
	var err error

	im.DiffIDs = im.generateDiffIds()
	im.ChainIDs = im.generateChainIds(im.DiffIDs)
	metaData, err := im.generateImageMetadata()
	if err != nil {
		return "", err
	}
	// imageID := im.writeImageMetadata(metaData)
	imageID := im.writeImageMetadata(metaData)
	if len(im.LayerPathsToSquash) != 0 {
		layerPathID, err = im.generateSquashedLayerPathId()
		if err != nil {
			return "", err
		}

		if im.OCIFormat {

			oldLayerPath = im.OldManifest.Config
		} else {

			if len(im.LayerPathsToSquash[0]) != 0 {
				oldLayerPath = im.LayerPathsToSquash[0]
			} else {
				oldLayerPath = layerPathID
			}
			oldLayerPath = filepath.Join(oldLayerPath, "json")

		}
		metaData, err = im.generateLastLayerMetadata(layerPathID, oldLayerPath)
		im.writeSquashedLayerMetadata(metaData)

		if err := im.writeVersionFile(im.SquashedDir); err != nil {
			return "", err
		}

		// 计算新的路径
		destPath := filepath.Join(im.NewImageDir, layerPathID)

		// 移动目录
		err := os.Rename(im.SquashedDir, destPath)
		if err != nil {
			// log.Fatalf("Failed to move directory from %s to %s: %v", im.SquashedDir, destPath, err)
			im.Logger.Errorf("Failed to move directory from %s to %s: %v", im.SquashedDir, destPath, err)
			return "", err
		}

	}
	manifest := im.generateManifestMetadata(imageID, layerPathID)

	if err := im.writeManifestMetadata(manifest); err != nil {
		return "", err
	}

	layers := manifest.Layers

	repositoryImageId := strings.Split(layers[len(layers)-1], "/")[0]

	if err := im.moveLayers(); err != nil {
		return "", err
	}

	repositoriesFile := filepath.Join(im.NewImageDir, "repositories")
	im.generateRepositoriesJson(repositoriesFile, repositoryImageId)

	return imageID, nil

}

func (im *V2Image) squashLayers() error {
	// 初始化要合并的层和要移动的层
	layersToSquash := im.LayerPathsToSquash
	// layersToMove := im.LayerPathsToMove
	oldImageDir := im.OldImageDir
	squashedTarPath := im.SquashedTar
	ociFormat := im.OCIFormat

	fmt.Printf("Starting squashing for %s...\n", squashedTarPath)

	// 反转需要合并的层的顺序
	ReverseList(layersToSquash)

	// 创建存放合并层的 tar 文件
	squashedFile, err := os.Create(squashedTarPath)
	if err != nil {
		return err
	}
	defer squashedFile.Close()

	squashedTar := tar.NewWriter(squashedFile)
	defer squashedTar.Close()

	// 初始化变量
	toSkip := []string{}
	squashedFiles := map[string]bool{}
	opaqueDirs := []string{}

	for _, layerID := range layersToSquash {
		layerTarFile := filepath.Join(oldImageDir, layerID)
		if !ociFormat {
			layerTarFile = filepath.Join(layerTarFile, "layer.tar")
		}
		fmt.Printf("Squashing file '%s'...\n", layerTarFile)

		layerTarReader, layerFile, err := openTarFile(layerTarFile)
		if err != nil {
			return err
		}
		defer layerFile.Close()

		layerOpaqueDirs := []string{}

		for {
			header, err := layerTarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			normalizedName := normPath(header.Name)
			if strings.Contains(header.Name, ".wh.") {
				if strings.HasSuffix(header.Name, ".wh..wh..opq") {
					opaqueDir := filepath.Dir(header.Name)
					layerOpaqueDirs = append(layerOpaqueDirs, opaqueDir)
					if !anyPrefix(opaqueDirs, opaqueDir) {
						if err := squashedTar.WriteHeader(header); err != nil {
							return err
						}
						if _, err := io.Copy(squashedTar, layerTarReader); err != nil {
							return err
						}
					}
				} else {
					toSkip = append(toSkip, normPath(strings.Replace(header.Name, ".wh.", "", 1)))
				}
				continue
			}

			if anyPrefix(opaqueDirs, normalizedName) {
				continue
			}

			if anyPrefix(toSkip, normalizedName) {
				continue
			}

			if squashedFiles[normalizedName] {
				continue
			}

			if err := squashedTar.WriteHeader(header); err != nil {
				return err
			}
			if header.Typeflag == tar.TypeReg {
				if _, err := io.Copy(squashedTar, layerTarReader); err != nil {
					return err
				}
			}
			squashedFiles[normalizedName] = true
		}

		opaqueDirs = append(opaqueDirs, layerOpaqueDirs...)
	}

	fmt.Println("Squashing finishing!")
	return nil
}

// 帮助函数，用于处理路径、打开 tar 文件等
func normPath(name string) string {
	return filepath.Clean("/" + name)
}

func openTarFile(filePath string) (*tar.Reader, *os.File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	tarReader := tar.NewReader(file)
	return tarReader, file, nil
}

func anyPrefix(slice []string, prefix string) bool {
	for _, s := range slice {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func (im *V2Image) writeVersionFile(squashedDir string) error {
	versionFile := filepath.Join(squashedDir, "VERSION")

	// Open the file for writing, create it if not exist, and truncate it if it exists
	file, err := os.OpenFile(versionFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Write the version number to the file
	if _, err := file.WriteString("1.0"); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

func (im *V2Image) writeSquashedLayerMetadata(metaData *ImageConfig) {

	layerMetadataFile := filepath.Join(im.SquashedDir, "json")

	// jsonMetadata, _ := im.dumpJson(metaData, false)
	jsonData, err := json.Marshal(metaData)
	if err != nil {
		panic(err) // handle the error appropriately in production code
	}

	im.writeJsonMetadata(string(jsonData), layerMetadataFile)

}

func (im *V2Image) generateRepositoriesJson(repositoriesFile, repositoryImageId string) error {
	if len(repositoryImageId) == 0 {
		return fmt.Errorf("Provided image id cannot be null")
	}
	if len(im.ImageName) == 0 && len(im.ImageTag) == 0 {
		return fmt.Errorf("No name and tag provided for the image, skipping generating repositories file")

	}
	repos := make(map[string]map[string]string)
	repos[im.ImageName] = make(map[string]string)
	repos[im.ImageName][im.ImageTag] = repositoryImageId

	// Marshal the data to JSON with compact formatting
	data, err := json.Marshal(repos)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	// Open the file for writing, create it if not exist, truncate if it exists
	file, err := os.OpenFile(repositoriesFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Write JSON data to the file
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	// Write a newline at the end of the file
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("error writing newline to file: %w", err)
	}

	return nil

}

func (im *V2Image) moveLayers() error {
	for _, layer := range im.LayerPathsToMove {
		layerID := strings.Replace(layer, "sha256:", "", -1)
		im.Logger.Debugf("Moving unmodified layer '%s'...", layerID)
		srcPath := filepath.Join(im.OldImageDir, layerID)
		prefix := "blobs/sha256/"
		destPath := filepath.Join(im.NewImageDir, layerID[len(prefix):])

		// Move the layer from src to dest
		if err := os.Rename(srcPath, destPath); err != nil {
			// Handle the case where the destination might be on a different filesystem
			return fmt.Errorf("failed to move layer '%s': %w", layerID, err)
		}
	}
	return nil
}

func (im *V2Image) writeManifestMetadata(manifest ImageManifest) error {

	manifestFile := filepath.Join(im.NewImageDir, "manifest.json")

	manifests := []ImageManifest{}
	// 打开一个文件用于写入

	manifests = append(manifests, manifest)
	file, err := os.Create(manifestFile)
	if err != nil {
		im.Logger.Errorf("Error creating file:", err)
		return err
	}
	defer file.Close()

	// 创建一个json.Encoder对象，并使用它来编码
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(manifests); err != nil {
		im.Logger.Errorf("Error encoding JSON:", err)
		return err
	}
	return nil

}

func (im *V2Image) generateManifestMetadata(imageID string, layerPathID string) ImageManifest {
	// manifest := make(map[string]interface{})

	manifest := ImageManifest{}
	manifest.Config = fmt.Sprintf("%s.json", imageID)
	if im.ImageName != "" && im.ImageTag != "" {
		manifest.RepoTags = []string{fmt.Sprintf("%s:%s", im.ImageName, im.ImageTag)}
	}

	var layers []string

	for _, layer := range im.OldManifest.Layers {
		layers = append(layers, layer)
	}

	if len(layers) > len(im.LayerPathsToMove) {
		layers = layers[:len(im.LayerPathsToMove)]
	}
	manifest.Layers = layers

	if layerPathID != "" {
		manifest.Layers = append(manifest.Layers, fmt.Sprintf("%s/layer.tar", layerPathID))
	}

	return manifest

}

func (im *V2Image) generateLastLayerMetadata(layerPathID, oldLayerPath string) (*ImageConfig, error) {
	configFilePath := filepath.Join(im.OldImageDir, oldLayerPath)

	// Read the JSON configuration file
	fileData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// var config map[string]interface{}
	imConfig := ImageConfig{}

	if err := json.Unmarshal(fileData, &imConfig); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	// Update the creation date
	imConfig.Created = im.Date.Format(time.RFC3339)

	// Update the image ID based on the squash ID condition
	if len(im.SquashID) != 0 {
		imConfig.Config.Image = im.SquashID
	}

	// Update 'parent' to the last layer to move, if available
	if len(im.LayerPathsToMove) > 0 {

		imConfig.Parent = im.LayerPathsToMove[len(im.LayerPathsToMove)-1]
	} else {
		imConfig.Parent = ""
	}

	// Update 'id' to the new layer path ID
	imConfig.ID = layerPathID

	// Remove 'container' field, if present
	imConfig.Container = ""

	return &imConfig, nil
}

func (im *V2Image) generateSquashedLayerPathId() (string, error) {

	// Copy and update the old image configuration
	v1Metadata := im.OldImageConfig

	// Update creation date

	v1Metadata.Created = im.Date.Format(time.RFC3339)

	v1Metadata.History = nil
	v1Metadata.Rootfs = Rootfs{}
	v1Metadata.Container = ""

	// Set 'layer_id' to the chain_id of the squashed layer
	if len(im.ChainIDs) > 0 {
		v1Metadata.LayerID = fmt.Sprintf("sha256:%s", im.ChainIDs[len(im.ChainIDs)-1])
	}

	// Handle 'parent'
	var parent string
	if len(im.LayerPathsToMove) > 0 {
		if len(im.LayerPathsToSquash) > 0 {
			parent = im.LayerPathsToMove[len(im.LayerPathsToMove)-1]
		} else {
			parent = im.LayerPathsToMove[0]
		}
		v1Metadata.Parent = fmt.Sprintf("sha256:%s", parent)
	}

	if len(im.SquashID) != 0 {
		v1Metadata.Config.Image = im.SquashID
	} else {
		v1Metadata.Config.Image = ""
	}

	jsonData, err := json.Marshal(v1Metadata)
	if err != nil {
		panic(err) // handle the error appropriately in production code
	}

	// Calculate the SHA256 hash of the JSON string
	hasher := sha256.New()
	hasher.Write([]byte(string(jsonData)))
	sha := fmt.Sprintf("%x", hasher.Sum(nil))

	return sha, nil
}

func (im *V2Image) writeImageMetadata(metaData *ImageConfig) string {

	// jsonMetadata, imageId := im.dumpJson(metadata, true)
	jsonData, err := json.Marshal(metaData)
	if err != nil {
		panic(err) // handle the error appropriately in production code
	}

	// Convert byte slice to string and optionally add newline
	jsonString := string(jsonData) + "\n"

	// Calculate the SHA256 hash of the JSON string
	hasher := sha256.New()
	hasher.Write([]byte(jsonString))
	imageId := fmt.Sprintf("%x", hasher.Sum(nil))

	imageMetadataFile := filepath.Join(im.NewImageDir, imageId+".json")
	if err := im.writeJsonMetadata(jsonString, imageMetadataFile); err != nil {
		im.Logger.Fatal("write metadata json failed")
		return ""
	}

	return imageId

}

func (im *V2Image) writeJsonMetadata(metadata string, metadataFile string) error {
	file, err := os.OpenFile(metadataFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err // return the error to be handled by the caller
	}
	defer file.Close() // ensure the file is closed after writing is done

	// Write the metadata to the file
	_, err = file.WriteString(metadata)
	if err != nil {
		return err // return the error to be handled by the caller
	}

	return nil // return nil on success

}

func (im *V2Image) generateImageMetadata() (*ImageConfig, error) {

	// Deep copy the old image config to metadata
	metadata := &ImageConfig{
		Architecture:  im.OldImageConfig.Architecture,
		Author:        im.OldImageConfig.Author,
		Config:        im.OldImageConfig.Config,
		DockerVersion: im.OldImageConfig.DockerVersion,
		OS:            im.OldImageConfig.OS,
		Rootfs:        im.OldImageConfig.Rootfs,
	}

	// Update image creation date
	metadata.Created = im.Date.Format(time.RFC3339)

	// Adjust history according to the layers to move
	if len(im.OldImageConfig.History) > len(im.LayersToMove) {
		metadata.History = im.OldImageConfig.History[:len(im.LayersToMove)]
	}

	if len(im.OldImageConfig.Rootfs.DiffIds) > len(im.LayerPathsToMove) {
		metadata.Rootfs.DiffIds = im.OldImageConfig.Rootfs.DiffIds[:len(im.LayerPathsToMove)]
	}

	historyItem := HistoryItem{

		Comment:   im.Comment,
		Created:   im.Date.Format(time.RFC3339),
		CreatedBy: im.LastCreatedBy,
	}

	// Handle layer paths to squash
	if len(im.LayerPathsToSquash) > 0 {

		metadata.Rootfs.DiffIds = append(metadata.Rootfs.DiffIds, fmt.Sprintf("sha256:%s", im.DiffIDs[len(im.DiffIDs)-1]))

	} else {
		historyItem.EmptyLayer = true
	}

	// Add new history entry
	metadata.History = append(metadata.History, historyItem)

	// Update image ID
	if len(im.SquashID) != 0 {
		metadata.Config.Image = im.SquashID
	} else {
		metadata.Config.Image = ""
	}

	return metadata, nil
}

func (im *V2Image) generateChainIds(diffIDs []string) []string {
	var chainIDs []string
	im.generateChainId(&chainIDs, diffIDs, "")
	return chainIDs
}

func (im *V2Image) generateChainId(chainIDs *[]string, diffIDs []string, parentChainID string) []string {
	if parentChainID == "" {
		return im.generateChainId(chainIDs, diffIDs[1:], diffIDs[0])
	}

	*chainIDs = append(*chainIDs, parentChainID)

	if len(diffIDs) == 0 {
		return []string{parentChainID}
	}

	toHash := fmt.Sprintf("sha256:%s sha256:%s", parentChainID, diffIDs[0])
	hasher := sha256.New()
	hasher.Write([]byte(toHash))
	digest := fmt.Sprintf("%x", hasher.Sum(nil))

	return im.generateChainId(chainIDs, diffIDs[1:], digest)
}

func (im *V2Image) extractTarName(path string) string {
	if im.OCIFormat {
		return filepath.Join(im.OldImageDir, path)
	}
	return filepath.Join(im.OldImageDir, path, "layer.tar")
}

func (im *V2Image) generateDiffIds() []string {
	var diffIDs []string

	for _, path := range im.LayerPathsToMove {
		layerTar := im.extractTarName(path)
		sha256, err := im.computeSha256(layerTar)
		if err != nil {
			panic(err) // Handle the error according to your application's requirements
		}
		diffIDs = append(diffIDs, sha256)
	}

	if len(im.LayerPathsToSquash) != 0 {
		sha256, err := im.computeSha256(filepath.Join(im.SquashedDir, "layer.tar"))
		if err != nil {
			panic(err) // Handle the error according to your application's requirements
		}
		diffIDs = append(diffIDs, sha256)
	}

	return diffIDs
}

func (im *V2Image) computeSha256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()

	// Read the file in chunks to avoid high memory consumption
	buffer := make([]byte, 10485760) // 10MB
	for {
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}
		if bytesRead == 0 {
			break
		}

		hasher.Write(buffer[:bytesRead])
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func (im *V2Image) isInOpaqueDirs(name string, opaqueDirs []string) bool {
	for _, opaqueDir := range opaqueDirs {
		if strings.HasPrefix(name, opaqueDir) {
			return true
		}
	}
	return false
}

func (im *V2Image) Cleanup() error {
	im.Logger.Infof("Cleaning up %s temporary directory", im.TmpDir)
	return os.RemoveAll(im.TmpDir)
}

func (im *V2Image) initializeDirectories() error {

	if err := im.prepareTmpDirectory(); err != nil {
		return err
	}

	// Temporary location on the disk of the old, unpacked *image*
	im.OldImageDir = filepath.Join(im.TmpDir, "old")
	// Temporary location on the disk of the new, unpacked, squashed *image*
	im.NewImageDir = filepath.Join(im.TmpDir, "new")
	// Temporary location on the disk of the squashed *layer*
	im.SquashedDir = filepath.Join(im.NewImageDir, "squashed")

	for _, dir := range []string{im.OldImageDir, im.NewImageDir} {
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

func (im *V2Image) squashId(layer string) (string, error) {
	if layer == "<missing>" {
		im.Logger.Info("You try to squash from layer that does not have it's own ID, we'll try to find it later")
	}

	imageInfo, _, err := im.DockerClient.ImageInspectWithRaw(context.Background(), layer)
	if err != nil {
		return "", err
	}
	im.Logger.Infof("Layer ID to squash from: %s", imageInfo.ID)
	return imageInfo.ID, nil
}

func (v2 *V2Image) validateNumberofLayers(number_of_layers int) error {
	//Makes sure that the specified number of layers to squash is a valid number

	if number_of_layers <= 0 {
		return fmt.Errorf("Number of layers to squash cannot be less or equal 0, provided: {%s}", string(number_of_layers))
	}
	if number_of_layers > len(v2.OldImageLayers) {
		return fmt.Errorf("Cannot squash {%s} layers, the {%s} image contains only {%s} layers", number_of_layers, v2.ImageName, len(v2.OldImageLayers))
	}

	return nil

}

func (im *V2Image) beforeSquashing() error {

	if err := im.initializeDirectories(); err != nil {
		return err
	}

	// Location of the tar archive with squashed layers
	im.SquashedTar = filepath.Join(im.SquashedDir, "layer.tar")

	if len(im.Tag) != 0 {
		im.parseImageName()
	}

	imageInfo, _, err := im.DockerClient.ImageInspectWithRaw(context.Background(), im.Image)
	if err != nil {
		im.Logger.Errorf("Could not get the image ID to squash, please check provided 'image' argument:", im.Image)
		return err
	}
	im.OldImageId = imageInfo.ID

	if err := im.readLayers(im.OldImageId); err != nil {
		return err
	}

	ReverseList(im.OldImageLayers)
	im.Logger.Infof("Old image has %d layers", len(im.OldImageLayers))
	im.Logger.Debugf("Old layers: %s", im.OldImageLayers)

	numOfLayers, err := strconv.Atoi(im.FromLayer)

	if err == nil {
		im.Logger.Debug("We detected number of layers as the argument to squash")
	} else {
		im.Logger.Debug("We detected layer as the argument to squash")
		squashId, err := im.squashId(im.FromLayer)
		if err != nil || len(squashId) == 0 {
			im.Logger.Infof("The %s layer could not be found in the %s image", im.FromLayer, im.Image)
			return err
		}
		numOfLayers = len(im.OldImageLayers) - FindIndex(im.OldImageLayers, squashId) - 1

	}

	if err := im.validateNumberofLayers(numOfLayers); err != nil {
		return err
	}

	maker := len(im.OldImageLayers) - numOfLayers

	im.LayersToSquash = im.OldImageLayers[maker:]
	im.LayersToMove = im.OldImageLayers[:maker]

	im.Logger.Info("Checking if squashing is necessary...")

	if len(im.LayersToSquash) < 1 {
		return fmt.Errorf("Invalid number of layers to squash:  %s", len(im.LayersToSquash))
	}
	if len(im.LayersToSquash) == 1 {
		return fmt.Errorf("Single layer marked to squash, no squashing is required")
	}
	im.Logger.Infof("Attempting to squash last [ %d ] layers...", numOfLayers)

	im.Logger.Debugf("Layers to squash: {%s}", im.LayersToSquash)
	im.Logger.Debugf("Layers to move: {%s}", im.LayersToMove)

	if err := im.saveImage(); err != nil {
		return err
	}
	im.SizeBefore, err = im.dirSize(im.OldImageDir)
	if err != nil {
		return err
	}
	im.Logger.Infof("Squashing image '%s'...", im.Image)

	if err := im.getManifest(); err != nil {
		return err
	}
	im.Logger.Debugf("Retrieved manifest '%s' ", im.OldManifest)

	if err := im.getIamgeConfig(); err != nil {
		return err
	}

	if err := im.readLayerPaths(); err != nil {
		return err
	}

	if len(im.LayerPathsToMove) > 0 {
		im.SquashID = im.LayerPathsToMove[len(im.LayerPathsToMove)-1]
	}
	im.Logger.Debugf("Layers paths to squash: %s", im.LayerPathsToSquash)
	im.Logger.Debugf("Layers paths to move: %s", im.LayerPathsToMove)
	return nil

}

func (v2 *V2Image) readLayerPaths() error {

	var currentManifestLayer int

	for i, layer := range v2.OldImageConfig.History {
		if layer.EmptyLayer == false { // Check if the layer is not empty
			var layerID string

			layers := v2.OldManifest.Layers

			if v2.OCIFormat {
				layerID = layers[currentManifestLayer]
			} else {
				layerID = strings.Split(layers[currentManifestLayer], "/")[0]
			}

			// Determine whether to move or squash this layer
			if len(v2.LayersToMove) > i {
				v2.LayerPathsToMove = append(v2.LayerPathsToMove, layerID)
			} else {
				v2.LayerPathsToSquash = append(v2.LayerPathsToSquash, layerID)
			}

			currentManifestLayer += 1
		}
	}

	return nil

}

// getManifest checks for the presence of "index.json" to determine if it's an OCI format.
// tries to load "manifest.json".

func (v2 *V2Image) getManifest() error {
	indexPath := filepath.Join(v2.OldImageDir, "index.json")
	manifestPath := filepath.Join(v2.OldImageDir, "manifest.json")

	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {

		v2.OCIFormat = true
	}

	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	var manifest []ImageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if len(manifest) == 0 {
		return errors.New("manifest is empty")
	}
	v2.OldManifest = manifest[0]
	return nil

}

func (v2 *V2Image) getIamgeConfig() error {

	configPath := filepath.Join(v2.OldImageDir, v2.OldManifest.Config)

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	if err := json.Unmarshal(data, &v2.OldImageConfig); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	return nil
}

func (v2 *V2Image) dirSize(directory string) (int64, error) {

	var size int64

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only include regular files in the size calculation.
		if !info.IsDir() {
			size += info.Size()
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("error walking the directory tree: %v", err)
	}

	return size, nil
}

func (im *V2Image) saveImage() error {
	//Saves the image as a tar archive under specified name

	for i := 0; i < 3; i++ {
		im.Logger.Infof("Saving image %s to %s directory...", im.OldImageId, im.OldImageDir)
		im.Logger.Infof("Try #%d...", (i + 1))

		reader, err := im.DockerClient.ImageSave(context.Background(), []string{im.OldImageId})
		if err != nil {
			im.Logger.Errorf("An error occurred while fetching the %s image, retrying: %v", im.OldImageId, err)
			continue
		}
		defer reader.Close()

		err = im.extractTar(reader, im.OldImageDir)
		if err == nil {
			im.Logger.Info("Image saved successfully!")
			return nil
		}

		im.Logger.Infof("An error occurred while extracting the %s image, retrying: %v", im.OldImageId, err)

	}
	return nil

}

// extractTar extracts a tar archive to a specified directory
func (v2 *V2Image) extractTar(tarReader io.Reader, directory string) error {
	tarBallReader := tar.NewReader(tarReader)

	for {
		header, err := tarBallReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar archive: %v", err)
		}

		path := fmt.Sprintf("%s/%s", directory, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("couldn't create directory: %v", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			file, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("couldn't create file: %v", err)
			}
			defer file.Close()

			if _, err := io.Copy(file, tarBallReader); err != nil {
				return fmt.Errorf("couldn't copy file contents: %v", err)
			}
			file.Chmod(os.FileMode(header.Mode))
		case tar.TypeLink:
			if err := os.Link(header.Linkname, path); err != nil {
				return fmt.Errorf("couldn't create hard link: %v", err)
			}
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, path); err != nil {
				return fmt.Errorf("couldn't create symlink: %v", err)
			}
		case tar.TypeChar:
			if err := syscall.Mknod(path, syscall.S_IFCHR|uint32(header.Mode), int(mkdev(header.Devmajor, header.Devminor))); err != nil {
				return fmt.Errorf("couldn't create character device: %v", err)
			}
		case tar.TypeBlock:
			if err := syscall.Mknod(path, syscall.S_IFBLK|uint32(header.Mode), int(mkdev(header.Devmajor, header.Devminor))); err != nil {
				return fmt.Errorf("couldn't create block device: %v", err)
			}
		case tar.TypeFifo:
			if err := syscall.Mknod(path, syscall.S_IFIFO|uint32(header.Mode), 0); err != nil {
				return fmt.Errorf("couldn't create fifo: %v", err)
			}
		default:
			v2.Logger.Infof("Ignoring unknown file type %c in %s", header.Typeflag, header.Name)
		}
	}

	return nil
}

func mkdev(major, minor int64) uint32 {
	return uint32((major << 8) | (minor & 0xff) | ((minor & 0xfff00) << 12))
}

func (v2 *V2Image) parseImageName() {
	//Parses the provided image name and splits it in the name and tag part, if possible. If no tag is provided  'latest' is used.
	// 检查镜像名称中是否包含":"
	colonIndex := strings.LastIndex(v2.Tag, ":")
	if colonIndex > -1 && !strings.Contains(v2.Tag[colonIndex:], "/") {
		// 如果":"后面没有"/"，则认为":"后面的是标签
		v2.ImageName = v2.Tag[:colonIndex]
		v2.ImageTag = v2.Tag[colonIndex+1:]
	} else {
		// 如果不包含":"或者":"后面直接跟了"/"，则使用默认标签"latest"
		v2.ImageName = v2.Tag
		v2.ImageTag = "latest"
	}

}

func (im *V2Image) prepareTmpDirectory() error {
	// Creates temporary directory that is used to work on layers

	if len(im.TmpDir) != 0 {
		if _, err := os.Stat(im.TmpDir); !os.IsNotExist(err) {
			// 如果目录已存在，则返回错误
			return fmt.Errorf("the '%s' directory already exists, please remove it before you proceed", im.TmpDir)
		}
		if err := os.MkdirAll(im.TmpDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create temporary directory: %v", err)
		}

	} else {
		tmpDir, err := os.MkdirTemp("", "docker-squash-")
		if err != nil {
			return err
		}
		im.TmpDir = tmpDir
	}
	im.Logger.Infof("Using %s as the temporary directory", im.TmpDir)
	return nil
}

func (im *V2Image) readLayers(imageID string) error {

	history, err := im.DockerClient.ImageHistory(context.Background(), imageID)
	if err != nil {
		return err
	}
	// 遍历镜像的历史记录，并输出每一层的ID
	count := 0
	for _, layer := range history {
		im.OldImageLayers = append(im.OldImageLayers, layer.ID)
		count++
	}

	if len(im.FromLayer) == 0 {
		im.FromLayer = fmt.Sprintf("%d", count)
	}

	return nil
}

func (im *V2Image) LoadSquashedImage() error {

	tarFile := filepath.Join(im.TmpDir, "image.tar")

	if err := im.tarImage(tarFile, im.NewImageDir); err != nil {
		im.Logger.Errorf("Error creating tar: %v\n", err)
		return err
	}

	file, err := os.Open(tarFile)
	if err != nil {
		im.Logger.Errorf("Error opening tar file: %v\n", err)
		return err
	}
	defer file.Close()
	defer os.Remove(tarFile)

	fmt.Printf("Loading squashed image -->[ %s:%s ]...\n", im.ImageName, im.ImageTag)
	response, err := im.DockerClient.ImageLoad(context.Background(), file, true)
	if err != nil {
		im.Logger.Errorf("Error loading image: %v\n", err)
		return err
	}
	defer response.Body.Close()

	// Print the API response
	bodyBytes, _ := ioutil.ReadAll(response.Body)
	im.Logger.Debugf("Docker API Response:%s", string(bodyBytes))

	im.Logger.Info("Image loaded!")
	return nil

}

func (im *V2Image) tarImage(targetTarFile, directory string) error {
	file, err := os.Create(targetTarFile)
	if err != nil {
		return fmt.Errorf("error creating tar file: %v", err)
	}
	defer file.Close()

	tw := tar.NewWriter(file)
	defer tw.Close()

	return filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == directory {
			return nil // skip the root directory
		}

		relPath, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.Mode().IsDir() {
			data, err := os.Open(path)
			if err != nil {
				return err
			}
			defer data.Close()

			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})
}

func (im *V2Image) ExportTarArchive(outputPath string) error {

	if err := im.tarImage(outputPath, im.NewImageDir); err != nil {
		return err
	}

	im.Logger.Infof("Image available at '%s'", outputPath)

	return nil
}