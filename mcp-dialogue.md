The current project allows usage and configuration of mcp tools. I want to add a system of
interactive dialogs that allows users to do the following:

1. see a list of mcp servers
2. add/remove/restart an mcp server from the list (if adding, prompt user for which config
file location, from the list of standard crush.json locations)
3. select & view an mcp server details
3.1 see a list of available tools
3.1.1 select a specific tool
3.1.2 view tool schema
3.2 list of prompts provided by tool
3.3 list of resources provided by tool
3.2 view a list of prompts
3.3 an option to view invocation logs
3.3.1 we will need to add specific, always-on logging for invocation of an mcp server
3.3.2 logs should be written to the standard logging dir, but in a server-specific file
3.3.3 logs should include a session id - when viewing logs I should be able to filter by
session, by selecting from a list of sessions

Implement this feature, including comprehensive e2e tests
