package featureflag

const PKGConfig = "PKG_CONFIG" // DEVBOX_FEATURE_PKG_CONFIG

func init() {
	disabled(PKGConfig)
}
