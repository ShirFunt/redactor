# API Traffic Log Redactor

A small Go tool that reads HTTP traffic logs (NDJSON, one transaction per line) and redacts sensitive data before the logs are stored. All behavior is driven by a YAML rules file (`rules.yaml`), not by code.

## Build, run and test

Requires the Go version listed in `go.mod`. The only dependency is `gopkg.in/yaml.v3`.

    go build -o redactor .
    ./redactor --rules rules.yaml input.ndjson > output.ndjson
    ./redactor < input.ndjson > output.ndjson      # stdin also works
    go test ./...

On Windows the binary is `redactor.exe`.

- Redacted NDJSON goes to **stdout**.
- A per-rule redaction summary and warnings go to **stderr**.
- If the rules file cannot be loaded, the tool exits with code 1 and writes nothing to stdout.

./redactor --serve --addr :8080 --rules /etc/redactor/rules.yaml
curl -X POST --data-binary @input.ndjson http://localhost:8080/redact

## Design decisions and trade-offs

**Streaming.** Input is read one line at a time with `bufio.Reader` and written through a buffered writer. Memory is O(length of the longest line), not O(input size). Time is linear in the input size times the number of rules. I used `Reader` instead of `Scanner` because `Scanner` fails on lines over 64 KB. A single huge line is still loaded fully, which is a known limit.

**Recursion.** Each line is decoded into generic Go values and walked recursively, so depth does not matter. Numbers are decoded with `UseNumber`, so large numbers and `10.50` keep their exact text. Strings that themselves contain a JSON object or array (an escaped body, for example) are parsed and walked as well. If the text is not valid JSON, it is handled as plain text.

**Rule matching.**
- *Key rules* are checked first. If a key matches, the rule handles the whole value, and value rules do not run on it. Header names are matched case-insensitively. `scope` decides whether a rule applies to headers, the body, or both.
- *Value rules* then run on every string, in the order they appear in the config, each one on the output of the previous.
- `key=value` pairs inside strings (form-encoded bodies, query strings in `path`, free text such as `password=...`) are checked against the key rules. Other values are URL-decoded first, so `bob%40example.com` is recognized as an email.
- Empty values (`null`, `""`, `[]`, booleans) are left alone. They carry no secret, and this keeps the information that a field was empty.
- Card numbers must pass the Luhn check, so a random 16-digit number is not redacted.

**Failure handling.** A line that cannot be parsed (truncated JSON, plain text, trailing garbage) cannot be redacted safely, so its content is never written anywhere. The output gets a valid placeholder instead, `{"line":20,"redactor_error":"unparseable_line"}`. Stderr gets a warning with only the **line number**, never the content or the parser's error text, because both can quote the input. Blank lines are skipped. I chose a placeholder over dropping the line so that line numbers stay traceable and downstream consumers still see one record per input line. The placeholder is valid JSON with no sensitive data, so a second run leaves it unchanged.

**Idempotency.** Running the tool on its own output gives identical output. `mask` and `partial` produce values that do not match any rule again. `hash` leaves a value alone if it already looks like a SHA-256 hex digest. The placeholder is written with its keys in sorted order. A test runs the provided `input.ndjson` twice and compares the results.

## The JWT report

The provided pattern for JWTs used the standard base64 alphabet (`A-Za-z0-9+/=`). JWTs use **base64url**, which replaces `+` and `/` with `-` and `_` and has no padding. As a result:
- a token with `-` or `_` in its header or payload did not match at all, and leaked completely;
- a token with `_` only in its signature matched only up to that character, so the tail of the signature stayed visible.

I added a failing regression test that covers both shapes (see the commit history: the failing test comes first, then the fix). The fix changes the character classes to `[A-Za-z0-9_-]`.

Running the tool on the sample data also showed two more gaps in the provided rules:
- The card pattern only matched unbroken digits, so numbers like `4111 1111 1111 1111` or `5555-5555-5555-4444` leaked. I widened the pattern and kept Luhn as the filter.
- There are no rules for things like `ssn` and `phone` in the sample data. They are outside the required minimum, so I did not add them, but they would be easy to add as value rules.

## Deployment files

- **Dockerfile:** multi-stage build. The final image is `distroless/static:nonroot`, so it has no shell and runs as a non-root user. The binary is static (`CGO_ENABLED=0`).
- **k8s.yaml:** a Deployment with two replicas and a Service. The readiness probe keeps traffic away until the rules are loaded. The liveness probe restarts a hung process. The container uses a read-only root filesystem, drops all capabilities, and has resource limits.
- **Rules as a ConfigMap:** the rules are not baked into the image. They are mounted from a ConfigMap, so they can change without a rebuild. If the ConfigMap is missing, the pod does not start, which matches the fail-closed rule.
- **Not tested on a cluster.** I validated the manifest with `kubectl apply --dry-run=client`. The image tag `0.1.0` is a placeholder for a real release process.

./redactor --serve --addr :8080 --rules /etc/redactor/rules.yaml
curl -X POST --data-binary @input.ndjson http://localhost:8080/redact
## Known limitations and what I would do with more time

- **Key order and duplicate keys.** Decoded objects are Go maps, so output keys are sorted alphabetically and duplicate keys keep only the last value. The output is equivalent but not byte-identical to the input. A token-based rewriter would avoid this.
- **Unsalted SHA-256** of low-entropy values such as emails can be reversed by guessing. A keyed hash (HMAC with a secret) would be better.
- **Numbers are not scanned** by value rules. A card number stored as a JSON number under an unknown key is not detected.
- **Pair detection depends on key names.** `password=` is caught by the key rules, but a name like `access_token=` is not, unless the rules list it.
- **Card pattern limit.** It accepts 13 to 19 digits with single spaces or dashes. Two card numbers written directly next to each other could be read as one match, fail Luhn, and stay visible.
- **Invalid UTF-8** in the input is replaced by U+FFFD when decoded, so binary-looking data is not preserved byte for byte.
- **No performance measurements yet.** I did not benchmark the tool.
- With more time I would add benchmarks, a fuzz test for the parser, and more rule types.

## Inline on live production traffic with a sub-millisecond budget

I would stop decoding every request into generic Go values and re-encoding it, because that allocates heavily. Instead I would scan the bytes with a streaming tokenizer and rewrite only the parts that need redaction, in place. I would run cheap pre-checks (is there an `@`, the text `eyJ`, or a run of digits?) so that the regexes run only on the few strings that might match, and I would merge the rules into one pass. I would reuse buffers with `sync.Pool` and avoid per-request allocation. I would also decide explicitly whether the system fails closed (block traffic if redaction fails) or fails open, and keep the heavy work, such as hashing, out of the critical path, for example by redacting a copy of the traffic asynchronously in a sidecar rather than in the request path.