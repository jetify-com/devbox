package featureflag

// ResolveV2 uses the /v2/resolve endpoint when resolving packages.
var ResolveV2 = disable("RESOLVE_V2")
