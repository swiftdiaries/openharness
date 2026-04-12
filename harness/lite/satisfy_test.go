package lite

import (
	"github.com/swiftdiaries/openharness/harness"
)

// Compile-time interface satisfaction checks for all Lite implementations.
var (
	_ harness.HarnessRunner = (*LiteRunner)(nil)
	_ harness.AgentStore    = (*LiteAgentStore)(nil)
	_ harness.SecretStore   = (*LiteSecretStore)(nil)
	_ harness.ArtifactStore = (*LiteArtifactStore)(nil)
	_ harness.ToolRegistry  = (*LiteToolRegistry)(nil)
	_ harness.SkillStore    = (*LiteSkillStore)(nil)
	_ harness.ChannelRouter = (*LiteChannelRouter)(nil)
)
