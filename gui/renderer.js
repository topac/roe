// This file is required by the index.html file and will
// be executed in the renderer process for that window.
// No Node.js APIs are available in this process because
// `nodeIntegration` is turned off. Use `preload.js` to
// selectively enable features needed in the rendering
// process.

class Worker {
  constructor() {
    this._reset();
    this.onstop = () => {};

    // register the callback of the "roecli-ack" msg (send when an execution of roecli is done)
    nodeapis.ipcRenderer.removeAllListeners("roecli-ack");
    nodeapis.ipcRenderer.on("roecli-ack", (event, resp) => {
      if (resp.error || this._work()) {
        this.onstop(resp);
        this._reset();
      }
    })
  }

  _reset() {
    this.queue = [];
    this.running = false;
  }

  _work() {
    if (this.queue.length === 0) {
      return true;
    } else {
      nodeapis.ipcRenderer.send("roecli", this.queue.pop());
      return false;
    }
  }

  start({action, input, output, pass, recursive}) {
    if (this.running) {
      return;
    }

    this.running = true;

    // build the work queue
    this.queue = input.map(inputPath => {
      let args = [
        action === 'encrypt' ? "-encrypt" : "-decrypt",
        "-outdir",
        output,
        "-p",
        pass,
        recursive ? "-recursive" : null,
        inputPath
      ];

      args = args.filter(arg => arg !== null);

      return args;
    });

    // work the first element
    this._work();
  }
}

// the poor's man react component
class App {
  constructor() {
    const initialState = {
      input: [],
      output: "",
      pass: "",
      passConf: "",
      action: 'encrypt',
      recursive: false,
      running: false,
    }

    // get a ref to the html elements

    this.elements = {
      action: document.getElementById("params-action"),
      input: document.getElementById("params-input"),
      inputBtn: document.getElementById("params-input-btn"),
      inputDirBtn: document.getElementById("params-input-dir-btn"),
      outdir: document.getElementById("params-outdir"),
      outdirBtn: document.getElementById("params-outdir-btn"),
      pass: document.getElementById("params-pass"),
      passConf: document.getElementById("params-pass-conf"),
      sendBtn: document.getElementById("params-send-btn"),
      busy: document.getElementById("params-busy"),
      actionMsg: document.getElementById("action-msg"),
      worker: new Worker(),
    }

    // bind javascript callbacks

    this.elements.action.onchange = (e) => this.setState({
      ...initialState,
      action: e.target.value
    });

    this.elements.pass.onkeyup = (event) => this.setState({
      pass: this.elements.pass.value
    });
  
    this.elements.passConf.onkeyup = (event) => this.setState({
      passConf: this.elements.passConf.value
    });
  
    this.elements.sendBtn.onclick = (event) => confirm("Continue?") && this.setState({
      running: true
    });

    this.elements.worker.onstop = (resp) => {
      this.setState({
        running: false,
        pass: "",
        passConf: ""
      });
      
      let alertMsg = "Operation completed successfully";

      let trim = (str) => `${str}`.length > 200 ? `${str.substring(0, 197).trim()}...` : `${str}`.trim();

      if (resp.error) {
        alertMsg = `${resp.stdout}`.match(/failed to decrypt/) ? `Wrong decryption password.` : `An error occurred.`;
        alertMsg += `\n\n`;
        alertMsg += `message: ${trim(resp.error.message)}\n`;
        alertMsg += `stdout: ${trim(resp.stdout)}\n`;
        alertMsg += `stderr: ${trim(resp.stderr)}`;
      }

      setTimeout(() => alert(alertMsg), 100);
    }

    // set the intitial state

    this.state = {};
    this.setState(initialState);
  }

  // update the state object
  setState(newState) {
    let s = { ...this.state, ...newState };
    this.state = s;
    this.render();
  }

  // change the visual state of the elements according to the state object
  render() {
    const state = this.state;
    const setState = this.setState.bind(this);

    // update labels and placeholders
    this.elements.pass.placeholder = (state.action === 'encrypt') ? "Encryption password" : "Decryption password";
    this.elements.sendBtn.innerText = (state.action === 'encrypt') ? "Encrypt" : "Decrypt";
    this.elements.actionMsg.innerText = (state.action === 'encrypt') ? 
      "Encrypt any file (or a folder recursively) into a valid .bmp image. Big files are splitted." : 
      "Decrypt a .bmp image (or a folder recursively) back to the original file.";
    this.elements.inputBtn.innerText = (state.action === 'encrypt') ? 
      "Select files..." :
      "Select .bmp images...";

    // busy or not?
    [this.elements.pass, this.elements.passConf].forEach(e => e.disabled = state.running);
    [this.elements.inputBtn, this.elements.outdirBtn, this.elements.sendBtn].forEach(e => e.style.pointerEvents = state.running ? "none" : "auto");
    this.elements.busy.style.display = state.running ? "block" : "none";
    let select = M.FormSelect.getInstance(this.elements.action);
    select.input.disabled = state.running;
    select.wrapper.disabled = state.running;

    // the main button is enabled?
    let ready = (state.pass && state.pass === state.passConf && state.input.length > 0 && state.output);
    this.elements.sendBtn.disabled = (state.running || !ready) ? true : false;

    // set the values of all the input fields
    this.elements.pass.value = state.pass;
    this.elements.passConf.value = state.passConf;
    this.elements.input.value = state.input.length > 1 ? `(${state.input.length} files)` : (state.input[0] || '');
    this.elements.outdir.value = state.output;

    // open file dialog to select input file(s)
    this.elements.inputBtn.onclick = function() {
      let filters = (state.action === 'encrypt') ? [{name: 'All Files', extensions: ['*']}] : [{name: 'Images', extensions: ['bmp']}]
      nodeapis.dialog.showOpenDialog({ properties: ['openFile', 'multiSelections'], filters: filters })
        .then((resp) => {
          if (resp.canceled) return;
          let filtered = resp.filePaths.filter(p => nodeapis.lstatSync(p).isFile());
          setState({input: filtered, recursive: false});
        })
    }

    // open file dialog to select input folder
    this.elements.inputDirBtn.onclick = function() {
      nodeapis.dialog.showOpenDialog({ properties: ['openDirectory'], filters: [] })
        .then((resp) => {
          if (resp.canceled) return;
          let filtered = resp.filePaths.filter(p => nodeapis.lstatSync(p).isDirectory());
          if (filtered.length !== 1) return;
          setState({input: filtered, recursive: true});
        })
    }

    // open directory dialog to select output folder
    this.elements.outdirBtn.onclick = function() {
      nodeapis.dialog.showOpenDialog({ properties: ['openDirectory'] })
        .then((resp) => {
          if (resp.canceled) return;
          let filtered = resp.filePaths.filter(p => nodeapis.lstatSync(p).isDirectory());
          if (filtered.length !== 1) return;
          setState({output: filtered});
        })
    }

    // control the worker
    if (state.running) {
      this.elements.worker.start(state);
    }
  }
}

window.onload = function(){  
  M.AutoInit();
  new App();
}
