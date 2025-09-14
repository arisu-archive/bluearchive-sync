package global

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strconv"

	"github.com/arisu-archive/assets-dumper/pkg/resourceapi"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/javax"
)

type DeviceVersionManager struct {
	device      DeviceManager
	assetClient resourceapi.Client
}

func NewDeviceVersionManager(device DeviceManager, assetClient resourceapi.Client) *DeviceVersionManager {
	return &DeviceVersionManager{
		device:      device,
		assetClient: assetClient,
	}
}

func (v *DeviceVersionManager) GetCurrentVersions() (map[string]int64, error) {
	reader, err := v.device.PullFile(path.Join(AndroidDataPath, "Patch", "patch.version.map"))
	if err != nil {
		return nil, fmt.Errorf("failed to pull version map: %w", err)
	}
	defer reader.Close()

	buffer := bytes.NewBuffer(nil)
	if _, err := buffer.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("failed to read version data: %w", err)
	}

	decoder, err := javax.NewDecoder(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	versionMap, err := decoder.ReadObject()
	if err != nil {
		return nil, fmt.Errorf("failed to decode version map: %w", err)
	}

	hashmap, ok := versionMap.(*javax.HashMap)
	if !ok {
		return nil, fmt.Errorf("version map is not a HashMap")
	}

	versions := make(map[string]int64)
	for k, v := range hashmap.Data {
		versions[k.(string)] = int64(v.(int32))
	}

	return versions, nil
}

func (v *DeviceVersionManager) GetFileHashes() (map[string]string, error) {
	reader, err := v.device.PullFile(path.Join(AndroidDataPath, "Patch", "patch.file.hash"))
	if err != nil {
		return nil, fmt.Errorf("failed to pull file hash: %w", err)
	}
	defer reader.Close()

	buffer := bytes.NewBuffer(nil)
	if _, err := buffer.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("failed to read hash data: %w", err)
	}

	decoder, err := javax.NewDecoder(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	hashMap, err := decoder.ReadObject()
	if err != nil {
		return nil, fmt.Errorf("failed to decode hash map: %w", err)
	}

	hashmap, ok := hashMap.(*javax.HashMap)
	if !ok {
		return nil, fmt.Errorf("hash map is not a HashMap")
	}

	hashes := make(map[string]string)
	for k, v := range hashmap.Data {
		hashes[k.(string)] = v.(string)
	}

	return hashes, nil
}

func (v *DeviceVersionManager) UpdateVersions(versions map[string]int64) error {
	buffer, err := v.createVersionMapBuffer(versions)
	if err != nil {
		return fmt.Errorf("failed to create version buffer: %w", err)
	}

	if err := v.device.PushFile(buffer, path.Join(AndroidDataPath, "Patch", "patch.version.map"), 0o664); err != nil {
		return fmt.Errorf("failed to push version map: %w", err)
	}

	return nil
}

func (v *DeviceVersionManager) UpdateFileHashes(hashes map[string]string) error {
	buffer, err := v.createHashMapBuffer(hashes)
	if err != nil {
		return fmt.Errorf("failed to create hash buffer: %w", err)
	}

	if err := v.device.PushFile(buffer, path.Join(AndroidDataPath, "Patch", "patch.file.hash"), 0o664); err != nil {
		return fmt.Errorf("failed to push file hash: %w", err)
	}

	return nil
}

func (v *DeviceVersionManager) UpdateToLatestVersion(ctx context.Context) error {
	latestVersionString, err := v.assetClient.GetLatestPatchVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest patch version: %w", err)
	}

	latestVersion, err := strconv.ParseInt(latestVersionString, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse latest version: %w", err)
	}

	currentVersions, err := v.GetCurrentVersions()
	if err != nil {
		return fmt.Errorf("failed to get current versions: %w", err)
	}

	currentVersions["Preload"] = latestVersion
	return v.UpdateVersions(currentVersions)
}

func (v *DeviceVersionManager) createHashMapBuffer(data map[string]string) (*bytes.Buffer, error) {
	hashMap := javax.NewHashMap(make(map[any]any))
	for k, v := range data {
		hashMap.Set(k, v)
	}

	buffer := bytes.NewBuffer(nil)
	encoder, err := javax.NewEncoder(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}

	if err := encoder.WriteObject(hashMap); err != nil {
		return nil, fmt.Errorf("failed to write hash map: %w", err)
	}

	return buffer, nil
}

func (v *DeviceVersionManager) createVersionMapBuffer(versionMap map[string]int64) (*bytes.Buffer, error) {
	hashMap := javax.NewHashMap(make(map[any]any))
	for k, v := range versionMap {
		hashMap.Set(k, javax.Integer{Value: int32(v)})
	}

	buffer := bytes.NewBuffer(nil)
	encoder, err := javax.NewEncoder(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}

	if err := encoder.WriteObject(hashMap); err != nil {
		return nil, fmt.Errorf("failed to write version map: %w", err)
	}

	return buffer, nil
}
