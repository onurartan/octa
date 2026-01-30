/// &main.agentx
/// &file -> ./dashboard.js
/// AGENT -> TypescriptValidatorAgent
/// [in_prompt]: "Ensure that the Dashboard.js file complies with TypeScript rules. For example, display an error message when a variable defined as name = “example” is used as a number elsewhere. ^vscodeTool(ext:agentx)^ Allow me to see errors made while using the tool via VSCode. And most importantly, to see the tests, use `./static/js/dashboard.ts` to ensure the validation errors are correct. `^vscodeTool(ext:ms-vscode.vscode-typescript-next)^` Verify TypeScript syntax errors with the tool."