package paths

// TODO proper doc comments!

const ApiVersion = "3"
const PathPrefix = "/api/" + ApiVersion

const PathAuthToken = PathPrefix + "/auth/full"
const PathWallet = PathPrefix + "/wallet"
const PathRegister = PathPrefix + "/signup"
const PathPassword = PathPrefix + "/password"
const PathVerify = PathPrefix + "/verify"
const PathResendVerify = PathPrefix + "/verify/resend"
const PathClientSaltSeed = PathPrefix + "/client-salt-seed"

const PathUnknownEndpoint = PathPrefix + "/"
const PathWrongApiVersion = "/api/"

const PathPrometheus = "/metrics"
