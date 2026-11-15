package updates

// updatePublicKey verifies the signatures in each release's manifest.json.
// The matching private key is the UPDATE_SIGNING_KEY secret of the release
// environment; see cmd/release-manifest. Replacing this key means installed
// copies reject every release signed with the new one, so they must update
// by hand once.
const updatePublicKey = "sZjT8BB9/KG4JbQrZnvLJmAK13mfhRJ+ZcQCnFRmGRw="
