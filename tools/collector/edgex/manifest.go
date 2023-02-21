package edgex

type Manifest struct {
	Updated       string   `yaml:"updated"`
	Count         int      `yaml:"count"`
	LatestVersion string   `yaml:"latestVersion"`
	Versions      []string `yaml:"versions"`
}

func NewManifest() *Manifest {
	manifest := &Manifest{
		Updated:       "false",
		Count:         0,
		LatestVersion: "",
		Versions:      make([]string, 0),
	}
	return manifest
}
