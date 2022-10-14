package configs

import "github.com/spf13/viper"

type Config struct {
	_keyPrefix    string
	Model         string `toml:"model"`      // 运行环境(dev,test,prod)
	Port          int    `toml:"port"`       // gate监听端口
	GrpcPt        int    `toml:"grpcpt"`     // grpc监听端口
	GrpcIp        string `toml:"grpcip"`     // grpc作为gs时的ip
	PromethesPort int    `toml:"prometheus"` // prometheus端口
}

func NewDefaultConfig(prefix string) *Config {
	cfg := &Config{
		_keyPrefix:    prefix,
		Model:         "dev",
		Port:          1001,
		GrpcPt:        0,
		GrpcIp:        "",
		PromethesPort: 9090,
	}
	return cfg
}

func (cfg *Config) SetPrefix(prefix string) {
	cfg._keyPrefix = prefix
}

func (cfg *Config) SetDefaultConfi(vcfg *viper.Viper) {
	vcfg.SetDefault(cfg._keyPrefix+".model", cfg.Model)
	vcfg.SetDefault(cfg._keyPrefix+".port", cfg.Port)
	vcfg.SetDefault(cfg._keyPrefix+".grpcpt", cfg.GrpcPt)
	vcfg.SetDefault(cfg._keyPrefix+".grpcip", cfg.GrpcIp)
	vcfg.SetDefault(cfg._keyPrefix+".prometheus", cfg.PromethesPort)
}

func (cfg *Config) ReadConfig(vcfg *viper.Viper) {
	cfg.Port = vcfg.GetInt(cfg._keyPrefix + ".port")
	cfg.PromethesPort = vcfg.GetInt(cfg._keyPrefix + ".prometheus")
	cfg.Model = vcfg.GetString(cfg._keyPrefix + ".model")
	cfg.GrpcIp = vcfg.GetString(cfg._keyPrefix + ".grpcip")
	cfg.GrpcPt = vcfg.GetInt(cfg._keyPrefix + ".grpcpt")
	return
}

func getConfig(v *viper.Viper) (err error) {
	v.GetBool("")
	v.GetDuration("")

	v.GetString("")
	
	v.GetInt("")
	v.GetInt32("")
	v.GetInt64("")
	v.GetUint("")
	v.GetUint32("")
	v.GetUint64("")

	v.GetIntSlice("")
	v.GetStringSlice("")
	v.GetStringMapString("")      //map[string]string
	v.GetStringMapStringSlice("") //map[string][]string
	return
}
