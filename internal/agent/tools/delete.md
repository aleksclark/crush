Deletes files or directories from the filesystem.

<usage>
- Provide file_path to the file or directory to delete
- For non-empty directories, set recursive=true
- Tool requires permission before deletion
</usage>

<features>
- Deletes individual files
- Deletes empty directories
- Deletes non-empty directories recursively when recursive=true
- Saves file content to history before deletion (for undo tracking)
- Notifies LSP servers about file deletions
</features>

<limitations>
- Cannot delete files outside the working directory
- Requires explicit recursive=true for non-empty directories
- Deletion is permanent (though content is saved in session history)
</limitations>

<cross_platform>
- Works on Windows, macOS, and Linux
- Path separators handled automatically
</cross_platform>

<tips>
- Use View or LS tool first to verify the correct path
- For directories, check contents before deleting
- Set recursive=true only when you're certain
- Deleted file content is saved to session history for potential restoration
</tips>

<examples>
Delete a single file:
- file_path: "/path/to/file.txt"

Delete an empty directory:
- file_path: "/path/to/empty-dir"

Delete a directory and all its contents:
- file_path: "/path/to/directory"
- recursive: true
</examples>
