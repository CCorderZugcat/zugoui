import './style.css';
import './wasm/wasm_exec.js';
import wasmUrl from './client.wasm';

window.zugoui = new Promise(
    (res) => {
        globalThis.zugouiReady = res;
    }
);

async function run (...args) { 
    const go = new globalThis.Go();
    go.argv.push(...args);

    const result = await WebAssembly.instantiateStreaming(fetch(wasmUrl), go.importObject);
    go.run(result.instance);
}

(async () => {
    try {
        await run('anon/contact/app/rpc');
    } catch(err) {
        console.error(err);
    }
})()


