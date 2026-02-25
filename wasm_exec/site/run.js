async function run() {
    try {
        go = new Go();
        go.argv = goargs;

        result = await WebAssembly.instantiateStreaming(fetch("binary.wasm"), go.importObject);
        go.run(result.instance);
    } catch (error) {
        console.error(error.message);
        return 1;
    }
    return 0;
}
