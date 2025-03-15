

MoonBit CLI Tools:
https://www.moonbitlang.com/download/#moonbit-cli-tools

curl -fsSL https://cli.moonbitlang.com/install/unix.sh | bash

moon shell-completion --help

and install the VSCode extension


## Moonbit

```bash
moon new
```


```bash
Enter the path to create the project (. for current directory): hello-world
Select the create mode: exec
Enter your username: k33g_org
Enter your project name: hello
Enter your license: MIT
Initialized empty Git repository in /Users/k33g/Devs/LearningMoonbit/hello-world/.git/
```


```
cd hello-world
moon run ./src/main
```


## Extism

```bash
curl https://get.extism.org/cli | sh
```


```bash
extism call wasimancer-plugin-calc.wasm addNumbers \
  --input '{"a":30, "b":12}' \
  --log-level "info" \
  --wasi
echo ""
```



## Moonbit plugin for Extism

https://mooncakes.io/docs/#/extism/moonbit-pdk/

```bash
moon new greet
cd greet
# moon update
moon add extism/moonbit-pdk
rm -rf src/lib
```

Update the `main.mbt` file

```moonbit
pub fn greet() -> Int {
  let name = @host.input_string()
  let greeting = "Hello, \{name}!"
  @host.output_string(greeting)
  0 // success
}

fn main {

}
```

Update the `moon.pkg.json`

```json
{
  "import": [
    "extism/moonbit-pdk/pdk/host"
  ],
  "link": {
    "wasm": {
      "exports": [
        "greet"
      ],
      "export-memory-name": "memory"
    }
  }
}
```

extism call target/wasm/release/build/main/main.wasm greet --input "{\"name\":\"Bob\"}" --wasi

extism call target/wasm/release/build/main/main.wasm greet --input '{"name":"Bob"}' --wasi

moon build --release --target wasm 
cp target/wasm/release/build/main/main.wasm ./greetings.wasm
extism call greetings.wasm greet --input '{"name":"Bob Morane"}' --wasi