String.prototype.toBuffer = function (){
    var buf = new ArrayBuffer(this.length)
    var bufView = new Uint8Array(buf)         
    for (var i = 0, strLen = this.length; i < strLen; i++) {      
        bufView[i] = this.charCodeAt(i)       
    }
    return buf
}

// Grid.js
const Grid = tui.Grid;
Grid.applyTheme('clean', {
    outline: {
        border: '#515151'
    },
    area: {
        header: {
            background: '#515151',
            border: '#515151',
        },
        body: {
            background: '#404040',
        }
    },
    cell: {
        normal: {
            background: '#404040',
            showVerticalBorder: false,
            showHorizontalBorder: false,
            text: '#fff',
        },
        header: {
            background: '#515151',
            showVerticalBorder: false,
            showVerticalBorder: false,
            text: '#fff',
        },
        selectedHeader: {
            background: '#4444444',
            text: '#fff',
        }
    },
    scrollbar: {
        background: '#35363a',
        emptySpace: '#35363a',
        border: '#35363a',
    },
    row: {
        hover: {
            background: '#333333'
        }
    },
});

const grids = new Grid({
    el: document.getElementById("grid"),
    columns: [{
        name: '#',
        width: 60,
        minWidth: 60,
        resizable: true,
        sortable: true,
        sortingType: 'desc'
    }, {
        name: 'Host',
        width: 150,
        minWidth: 120,
        resizable: true,
        filter: 'select'
    }, {
        name: 'URL',
        minWidth: 200,
        resizable: true,
        filter: {
            type: 'text',
            showApplyBtn: true,
            showClearBtn: true
        }
    }, {
        name: 'Status',
        width: 80,
        minWidth: 80,
        resizable: true,
        // sortable: true
        align: 'center',
        filter: 'text'
    }, {
        name: 'Method',
        width: 80,
        minWidth: 80,
        resizable: true,
        // sortable: true
    }, {
        name: 'Length',
        width: 80,
        minWidth: 80,
        resizable: true,
        // sortable: true
    }],
    data: [],
    scrollX: true,
    scrollY: true,
    resizable: true,
    bodyHeight: 'fitToParent',
    // bodyHeight: 100,
    fixedHeader: true,
    selectionUnit: 'row',
    sortState: {
        columnName: '#', // The column to sort by default
        ascending: false     // Set to 'true' for ascending, 'false' for descending
    },
    style: {
        table: {
            'font-size': '13px',
            'overflow-y': 'hidden'
        },
        container: {
            'padding': '0px'
        }
    }
});

grids.sort('#', false, false)

var current = -1;
function isDecodeTargetMIME(content_type) {
    let MIME = ["text/", "application/json", "application/xml", "application/xhtml+xml"]
    let avoid = "javascript"
    for (let i = 0; i < MIME.length; i++) {
        if(content_type.indexOf(MIME[i]) !== -1 && content_type.indexOf(avoid) === -1) {
            return true
        }
    }
    return false
}

grids.on('focusChange', (ev) => {
    console.log(ev)
    const rowKey = grids.getRowCount() - ev.rowKey - 1
    grids.setSelectionRange({
        start: [rowKey, 0],
        end: [rowKey, grids.getColumns().length - 1]
    })

    current = ev.rowKey
    astilectron.sendMessage({
        type: "history",
        idx: current
    }, function (message) {
        console.log(message)
        if (message) {
            requestEditor.setValue(atob(message.request))
            let resp = atob(message.response)
            let tmp = resp.split("\n\n")
            let headerstr = tmp[0]
            let body = tmp.slice(1).join("\n\n")
            if (body.length >= 0) {
                let header = {}
                let reslines = headerstr.split("\n")
                reslines.slice(1).forEach((val) => {
                    let tmp = val.split(": ")
                    let key = tmp[0]
                    header[key] = tmp.slice(1).join(": ")
                })

                if (parseInt(header["Content-Length"]) > 0 && isDecodeTargetMIME(header["Content-Type"])) {
                    let decoder = new TextDecoder(Encoding.detect(resp))
                    resp = decoder.decode(resp.toBuffer())
                }
            }
            
            responseEditor.setValue(resp)
        }
    })
})

function Issue() {
    if (current < 0) {
        return
    }
    let request = requestEditor.getValue()
    let tmp = request.split("\n\n")
    let headerstr = tmp[0]
    let body = tmp.slice(1).join("\n\n")
    if (body.length > 0) {
        let header = {}
        let reqlines = headerstr.split("\n")
        reqlines.slice(1).forEach((val) => {
            let tmp = val.split(": ")
            let key = tmp[0]
            header[key] = tmp.slice(1).join(": ")
        })
        header["Content-Length"] = body.length
        let sheader = []
        for (let key in header) {
            sheader.push(key + ": " + header[key])
        }

        request = [reqlines[0], sheader.join("\n"), "", body].join("\n")
        requestEditor.setValue(request)
    }
    astilectron.sendMessage({
        type: "issue",
        idx: current,
        request: request
    }, function (message) {
        console.log(message)
        if (message) {
            responseEditor.setValue(atob(message.response))
        }
    })
}

window.addEventListener('keydown', function (event) {
    if (event.ctrlKey && event.code === 'KeyG') { // issue Ctrl + G
        Issue()
    }
});

CodeMirror.defineMode('http-json', function (config) {
    return CodeMirror.multiplexingMode(
        CodeMirror.getMode(config, 'http'),
        {
            open: /^\s*\{/, // JSON body를 시작하는 중괄호를 찾습니다.
            close: /^\s*\}/, // JSON body를 닫는 중괄호를 찾습니다.
            mode: CodeMirror.getMode(config, 'javascript'),
            delimStyle: 'delimit'
        }
    );
});

CodeMirror.defineMIME('http-json', 'http-json');

// CodeMirror
const requestEditor = CodeMirror(document.getElementById("request-editor"), {
    mode: "http-json",
    lineNumbers: true,
    lineWrapping: true,
    indentUnit: 4,
    theme: "dracula",
    extraKeys: {
        'Ctrl-E': function (editor) {
            editor.replaceSelection(encodeURIComponent(editor.getSelection()))
        },
        'Shift-Ctrl-E': function (editor) {
            editor.replaceSelection(decodeURIComponent(editor.getSelection()))
        },
        'Ctrl-B': function (editor) {
            editor.replaceSelection(btoa(editor.getSelection()))
        },
        'Shift-Ctrl-B': function (editor) {
            editor.replaceSelection(atob(editor.getSelection()))
        }
    }
});

const responseEditor = CodeMirror(document.getElementById("response-editor"), {
    mode: "http-json",
    lineNumbers: true,
    lineWrapping: true,
    indentUnit: 4,
    theme: "dracula",
    extraKeys: {
        'Ctrl-E': function (editor) {
            editor.replaceSelection(encodeURIComponent(editor.getSelection()))
        },
        'Shift-Ctrl-E': function (editor) {
            editor.replaceSelection(decodeURIComponent(editor.getSelection()))
        },
        'Ctrl-B': function (editor) {
            editor.replaceSelection(btoa(editor.getSelection()))
        },
        'Shift-Ctrl-B': function (editor) {
            editor.replaceSelection(atob(editor.getSelection()))
        }
    }
});

const leftPanel = document.querySelector(".left");
const resizer = document.querySelector('.middle');
resizer.addEventListener('mousedown', function (e) {
    e.preventDefault();
    document.addEventListener('mousemove', resize);
    document.addEventListener('mouseup', stopResize);
});

function resize(e) {
    console.log(e)
    leftPanel.style.flexBasis = e.clientX + 'px'
    grids.setWidth(e.clientX)
}

function stopResize() {
    document.removeEventListener('mousemove', resize);
}

document.addEventListener('astilectron-ready', function () {
    astilectron.onMessage(function (message) {
        console.log(message)
        if (message && message.type === "proxy") {
            const item = {
                '#': grids.getRowCount() + 1,
                Host: message.host,
                Status: message.status,
                Method: message.method,
                URL: message.url,
                Length: message.length
            }
            grids.appendRow(item)
            grids.setSelectionRange({
                start: [grids.getRowCount() - current - 1, 0],
                end: [grids.getRowCount() - current - 1, 5]
            })
        }
    })
})