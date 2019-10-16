// Modules to control application life and create native browser window
const {app, BrowserWindow, Menu, MenuItem, ipcMain } = require('electron')
const path = require('path')
const fs = require('fs')
const execFile = require('child_process').execFile;

function createWindow () {
  // Create the browser window.
  const mainWindow = new BrowserWindow({
    width: 600,
    minWidth: 600,
    maxWidth: 600,
    height: 500,
    minHeight: 500,
    icon: path.join(__dirname, 'assets/icons/icon256.png'),
    webPreferences: {
      preload: path.join(__dirname, 'preload.js')
    }
  })

  // and load the index.html of the app.
  mainWindow.loadFile('index.html')
  mainWindow.setMenuBarVisibility(false);

  let roeCliPath = getRoeCliPath();

  ipcMain.on("roecli", function(event, args) {
    if (!roeCliPath) {
      event.reply("roecli-ack", {error: "file not found", stdout: "", stderr: "cannot find roe-cli binary"});
      return
    }
    execFile(roeCliPath, args, function callback(error, stdout, stderr){
      event.reply("roecli-ack", {error, stdout, stderr});
    });
  })
}

function getRoeCliPath() {
  let paths = [
    path.join(__dirname, "extraResources", "roe-cli"),
    path.join(process.resourcesPath, "roe-cli"),
  ]

  for(let i = 0; i < paths.length; i++) {
    if (fs.existsSync(paths[i])) return paths[i];
  }
  
  return null;
}

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.whenReady().then(createWindow)

// Quit when all windows are closed.
app.on('window-all-closed', function () {
  // On macOS it is common for applications and their menu bar
  // to stay active until the user quits explicitly with Cmd + Q
  if (process.platform !== 'darwin') app.quit()
})

app.on('activate', function () {
  // On macOS it's common to re-create a window in the app when the
  // dock icon is clicked and there are no other windows open.
  if (BrowserWindow.getAllWindows().length === 0) createWindow()
})

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and require them here.
