String.prototype.toBuffer = function () {
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
        if (content_type.indexOf(MIME[i]) !== -1 && content_type.indexOf(avoid) === -1) {
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

CodeMirror.defineMode("httpWithContentType", function (config, parserConfig) {
    const httpMode = CodeMirror.getMode(config, "http");
    const jsonMode = CodeMirror.getMode(config, "javascript");
    const htmlmixedMode = CodeMirror.getMode(config, "htmlmixed");
    const cssMode = CodeMirror.getMode(config, "css");

    return {
        startState: function () {
            return {
                inBody: false,
                contentType: null,
                reqline: false,
                http: CodeMirror.startState(httpMode),
                json: CodeMirror.startState(jsonMode),
                htmlmixed: CodeMirror.startState(htmlmixedMode),
                css: CodeMirror.startState(cssMode),
            }
        },
        copyState: function (state) {
            return {
                inBody: state.inBody,
                contentType: state.contentType,
                http: CodeMirror.copyState(httpMode, state.http),
                json: CodeMirror.copyState(jsonMode, state.json),
                htmlmixed: CodeMirror.copyState(htmlmixedMode, state.htmlmixed),
                css: CodeMirror.copyState(cssMode, state.css),
            }
        },
        token: function (stream, state) {
            if (!state.inBody) {
                const token = httpMode.token(stream, state.http);
                if (token === "string" && stream.string.includes("Content-Type")) {
                    if (stream.string.includes("application/json")) {
                        state.contentType = "json";
                    } else if (stream.string.includes("html") || stream.string.includes("xml")) {
                        state.contentType = "htmlmixed";
                    } else if (stream.string.includes("css")) {
                        state.contentType = "css"
                    }
                }

                if (stream.eol()) {
                    if (token === null) {
                        if(state.reqline === false) {
                            state.reqline = true
                        } else {
                            state.inBody = true
                            stream.backUp(stream.pos)
                            return null
                        }
                    }
                }
                return token;
            }

            if (state.inBody) {
                if (state.contentType === "json") {
                    console.log("json")
                    return jsonMode.token(stream, state.json);
                } else if (state.contentType === "htmlmixed") {
                    console.log("htmlmixed")
                    return htmlmixedMode.token(stream, state.htmlmixed);
                } else if (state.contentType === "css") {
                    console.log("css~")
                    return cssMode.token(stream, state.css)
                }
            }

            return httpMode.token(stream, state.http);
        },
        innerMode: function (state) {
            if (!state.inBody) {
                return { state: state.http, mode: httpMode };
            } else {
                if (state.contentType === "json") {
                    return { state: state.json, mode: jsonMode };
                } else if (state.contentType === "htmlmixed") {
                    return { state: state.htmlmixed, mode: htmlmixedMode };
                } else if (state.contentType === "css") {
                    return { state: state.css, mode: cssMode };
                }
            }
        },
    };
});

// CodeMirror
const requestEditor = CodeMirror(document.getElementById("request-editor"), {
    mode: "httpWithContentType",
    lineNumbers: true,
    lineWrapping: true,
    indentUnit: 4,
    theme: "dracula",
    autoCloseBrackets: true,
    matchBrackets: true,
    styleActiveLine: true,
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
})

const responseEditor = CodeMirror(document.getElementById("response-editor"), {
    mode: "httpWithContentType",
    lineNumbers: true,
    lineWrapping: true,
    indentUnit: 4,
    theme: "dracula",
    autoCloseBrackets: true,
    matchBrackets: true,
    styleActiveLine: true,
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
        // console.log(message)
        if (message && message.type === "proxy") {
            async function add() {
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

            add()
        }
    })
})