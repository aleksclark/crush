package tools

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/filepathext"
	"github.com/charmbracelet/crush/internal/fsext"
	"github.com/charmbracelet/crush/internal/history"
	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/permission"
)

//go:embed delete.md
var deleteDescription []byte

// DeleteParams contains parameters for the delete tool.
type DeleteParams struct {
	FilePath  string `json:"file_path" description:"The absolute path to the file or directory to delete"`
	Recursive bool   `json:"recursive,omitempty" description:"If true and path is a directory, delete it and all its contents recursively. Required for non-empty directories."`
}

// DeletePermissionsParams contains parameters shown in the permission dialog.
type DeletePermissionsParams struct {
	FilePath  string `json:"file_path"`
	Recursive bool   `json:"recursive,omitempty"`
	IsDir     bool   `json:"is_dir,omitempty"`
}

// DeleteResponseMetadata contains metadata about the delete operation.
type DeleteResponseMetadata struct {
	FilePath string `json:"file_path"`
	IsDir    bool   `json:"is_dir"`
}

const DeleteToolName = "delete"

// NewDeleteTool creates a new delete tool instance.
func NewDeleteTool(lspClients *csync.Map[string, *lsp.Client], permissions permission.Service, files history.Service, workingDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		DeleteToolName,
		string(deleteDescription),
		func(ctx context.Context, params DeleteParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.FilePath == "" {
				return fantasy.NewTextErrorResponse("file_path is required"), nil
			}

			filePath := filepathext.SmartJoin(workingDir, params.FilePath)

			// Check if file/directory exists.
			fileInfo, err := os.Stat(filePath)
			if os.IsNotExist(err) {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("File or directory does not exist: %s", filePath)), nil
			}
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("error checking path: %w", err)
			}

			isDir := fileInfo.IsDir()

			// If it's a non-empty directory and recursive is false, return error.
			if isDir && !params.Recursive {
				entries, readErr := os.ReadDir(filePath)
				if readErr != nil {
					return fantasy.ToolResponse{}, fmt.Errorf("error reading directory: %w", readErr)
				}
				if len(entries) > 0 {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("Directory %s is not empty. Set recursive=true to delete non-empty directories.", filePath)), nil
				}
			}

			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("session_id is required")
			}

			// Request permission.
			p, err := permissions.Request(ctx,
				permission.CreatePermissionRequest{
					SessionID:   sessionID,
					Path:        fsext.PathOrPrefix(filePath, workingDir),
					ToolCallID:  call.ID,
					ToolName:    DeleteToolName,
					Action:      "delete",
					Description: fmt.Sprintf("Delete %s", filePath),
					Params: DeletePermissionsParams{
						FilePath:  filePath,
						Recursive: params.Recursive,
						IsDir:     isDir,
					},
				},
			)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}
			if !p {
				return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
			}

			// For files (not directories), save content to history before deletion.
			if !isDir {
				oldContent, readErr := os.ReadFile(filePath)
				if readErr == nil {
					// Check if file exists in history.
					file, histErr := files.GetByPathAndSession(ctx, filePath, sessionID)
					if histErr != nil {
						_, histErr = files.Create(ctx, sessionID, filePath, string(oldContent))
						if histErr != nil {
							slog.Error("Error creating file history before delete", "error", histErr)
						}
					} else if file.Content != string(oldContent) {
						// User manually changed the content; store an intermediate version.
						_, histErr = files.CreateVersion(ctx, sessionID, filePath, string(oldContent))
						if histErr != nil {
							slog.Error("Error creating file history version before delete", "error", histErr)
						}
					}
				}
			}

			// Close the file in LSP clients before deletion.
			if lspClients != nil {
				for client := range lspClients.Seq() {
					if err := client.CloseFile(ctx, filePath); err != nil {
						slog.Debug("Error closing file in LSP before delete", "file", filePath, "error", err)
					}
				}
			}

			// Perform the deletion.
			if isDir {
				if params.Recursive {
					err = os.RemoveAll(filePath)
				} else {
					err = os.Remove(filePath)
				}
			} else {
				err = os.Remove(filePath)
			}

			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("error deleting %s: %w", filePath, err)
			}

			// Notify LSP clients about the deletion.
			if lspClients != nil {
				for client := range lspClients.Seq() {
					if err := client.NotifyFileDeleted(ctx, filePath); err != nil {
						slog.Debug("Error notifying LSP of file deletion", "file", filePath, "error", err)
					}
				}
			}

			var result string
			if isDir {
				result = fmt.Sprintf("Directory successfully deleted: %s", filePath)
			} else {
				result = fmt.Sprintf("File successfully deleted: %s", filePath)
			}

			return fantasy.WithResponseMetadata(
				fantasy.NewTextResponse(fmt.Sprintf("<result>\n%s\n</result>", result)),
				DeleteResponseMetadata{
					FilePath: filePath,
					IsDir:    isDir,
				},
			), nil
		})
}
