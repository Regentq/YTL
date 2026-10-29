const fs = require('fs');
const path = require('path');
const vscode = require('vscode');
const { LanguageClient } = require('vscode-languageclient/node');

let client;

function activate(context) {
  const exe = process.platform === 'win32' ? 'ytl-lsp.exe' : 'ytl-lsp';
  const command = context.asAbsolutePath(path.join('bin', exe));

  if (!fs.existsSync(command)) {
    vscode.window.showErrorMessage(
      'YTL: не найден языковой сервер ' + command + '. Собери его командой go build.'
    );
    return;
  }

  const serverOptions = { command: command, args: [] };

  const clientOptions = {
    documentSelector: [
      { scheme: 'file', language: 'ytl' },
      { scheme: 'untitled', language: 'ytl' },
    ],
  };

  client = new LanguageClient('ytl', 'YTL Language Server', serverOptions, clientOptions);
  client.start();
}

function deactivate() {
  return client ? client.stop() : undefined;
}

module.exports = { activate, deactivate };