package skill

import (
	"context"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/AvatarGanymede/pcraft/internal/common/instructionrefs"
)

// deliver dispatches the manifest to the executor-specific strategy.
// Returns the metadata patches and instructions directory the caller
// should attach to the launch request.
func (d *Deployer) deliver(_ context.Context, manifest *Manifest, executorType, worktreePath string) DeployResult {
	if manifest == nil {
		return DeployResult{}
	}
	switch executorType {
	default:
		// local_pc: the worktree IS the agent's CWD, so writing
		// skills under <worktree>/<projectSkillDir>/kandev-<slug>
		// gets them in front of the agent's project-skill discovery.
		return d.deliverLocal(manifest, worktreePath)
	}
}

// deliverLocal writes skills directly into the session's worktree
// under the agent's project skill directory and writes instruction
// files to the host runtime tree. Used for local_pc executor —
// writes directly to the host filesystem.
func (d *Deployer) deliverLocal(manifest *Manifest, worktreePath string) DeployResult {
	if worktreePath != "" && manifest.ProjectSkillDir != "" {
		if err := injectSkills(worktreePath, manifest.ProjectSkillDir, manifest.Skills); err != nil {
			d.logger.Warn("failed to inject skills into worktree",
				zap.String("worktree", worktreePath),
				zap.String("dir", manifest.ProjectSkillDir),
				zap.Error(err))
		}
	}
	instructionsDir := instructionsDirHost(d.basePath, manifest.WorkspaceSlug, manifest.AgentID)
	d.writeInstructionFiles(manifest, instructionsDir)
	return DeployResult{InstructionsDir: instructionsDir}
}

// rewriteManifestRefs canonicalises sibling instruction references
// (./HEARTBEAT.md, ./SOUL.md, ...) inside each instruction file's
// content to absolute paths under instructionsDir. Used by the
// local writer so the contract matches the office prompt builder,
// which applies the same rewrite.
func rewriteManifestRefs(manifest *Manifest, instructionsDir string) {
	if manifest == nil || instructionsDir == "" {
		return
	}
	for i := range manifest.Instructions {
		manifest.Instructions[i].Content = instructionrefs.Rewrite(
			manifest.Instructions[i].Content, instructionsDir,
		)
	}
}

// writeInstructionFiles writes the manifest's instruction files into
// instructionsDir. Filenames that are not safe single-component
// strings are skipped to avoid path traversal. Sibling refs inside
// the file content are rewritten to absolute paths so the on-disk
// artefact agrees with the prompt the agent receives.
func (d *Deployer) writeInstructionFiles(manifest *Manifest, instructionsDir string) {
	if len(manifest.Instructions) == 0 {
		return
	}
	if err := os.MkdirAll(instructionsDir, 0o755); err != nil {
		d.logger.Warn("failed to create instructions dir", zap.Error(err))
		return
	}
	for _, instr := range manifest.Instructions {
		if !isValidPathComponent(instr.Filename) {
			d.logger.Warn("skipping instruction with invalid filename",
				zap.String("filename", instr.Filename))
			continue
		}
		content := instructionrefs.Rewrite(instr.Content, instructionsDir)
		if err := os.WriteFile(
			filepath.Join(instructionsDir, instr.Filename),
			[]byte(content), 0o644,
		); err != nil {
			d.logger.Warn("failed to write instruction file",
				zap.String("filename", instr.Filename), zap.Error(err))
		}
	}
}
