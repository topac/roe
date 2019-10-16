// All of the Node.js APIs are available in the preload process.
// It has the same sandbox as a Chrome extension.
window.addEventListener('DOMContentLoaded', () => {
  const { ipcRenderer } = require('electron');
  const { dialog } = require('electron').remote;

  window.nodeapis = {
    ipcRenderer, 
    dialog: dialog,
    lstatSync: require('fs').lstatSync
  };
})
