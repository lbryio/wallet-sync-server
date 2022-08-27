package paths

const ApiVersion = "3"
const PathPrefix = "/api/" + ApiVersion

const PathAuthToken = PathPrefix + "/auth/full"
const PathWallet = PathPrefix + "/wallet"
const PathRegister = PathPrefix + "/signup"
const PathPassword = PathPrefix + "/password"
const PathVerify = PathPrefix + "/verify"
const PathResendVerify = PathPrefix + "/verify/resend"
const PathClientSaltSeed = PathPrefix + "/client-salt-seed"

// Using such a generic name since, as I understand, we can do a bunch of
// different stuff over this one websocket.
const PathWebsocket = PathPrefix + "/websocket"

const PathUnknownEndpoint = PathPrefix + "/"
const PathWrongApiVersion = "/api/"

const PathPrometheus = "/metrics"
