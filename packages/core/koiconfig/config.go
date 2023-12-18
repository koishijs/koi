package koiconfig

type Config struct {
	Data     ConfigData
	Computed ConfigComputed
}

//goland:noinspection GoNameStartsWithPackageName
type ConfigData struct {
	Open    string `yaml:"open"`
	Isolate string `yaml:"isolate"`
	Start   []string
	Env     []string `yaml:"env"`
}

//goland:noinspection GoNameStartsWithPackageName
type ConfigComputed struct {
	Exe         string
	DirExe      string
	DirConfig   string
	DirData     string
	DirHome     string
	DirBin      string
	DirLock     string
	DirTemp     string
	DirInstance string
	DirLog      string
}
