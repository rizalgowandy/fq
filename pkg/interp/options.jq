include "internal";
include "binary";


def _opt_build_default_fixed:
  ( stdout_tty as $stdout
  | {
      addrbase:       16,
      arg:            [],
      argjson:        [],
      array_truncate: 50,
      bits_format:    "snippet",
      # 0-0xff=brightwhite,0=brightblack,32-126:9-13=white
      byte_colors:    [
        { ranges: [[0,255]],
          value: "brightwhite"
        },
        { ranges: [[0]],
          value: "brightblack"
        },
        { ranges: [[32,126],[9,13]],
          value: "white"
        }
      ],
      color:          ($stdout.is_terminal and (env.NO_COLOR | . == null or . == "")),
      colors: {
        null: "brightblack",
        false: "yellow",
        true: "yellow",
        number: "cyan",
        string: "green",
        objectkey: "brightblue",
        array: "white",
        object: "white",
        index: "white",
        value: "white",
        error: "brightred",
        dumpheader: "yellow+underline",
        dumpaddr: "yellow",
        prompt_repl_level: "brightblack",
        prompt_value: "white"
      },
      compact:            false,
      completion_timeout: (env.COMPLETION_TIMEOUT | if . != null then tonumber else 1 end),
      decode_file:        [],
      decode_format:      "probe",
      decode_progress:    (env.NO_DECODE_PROGRESS == null),
      depth:              0,
      expr:               ".",
      expr_eval_path:     "arg",
      expr_file:          null,
      filenames:          null,
      force:              false,
      include_path:       null,
      join_string:        "\n",
      null_input:         false,
      raw_file:            [],
      raw_output:         ($stdout.is_terminal | not),
      raw_string:         false,
      repl:               false,
      sizebase:           10,
      show_formats:       false,
      show_help:          false,
      slurp:              false,
      string_input:       false,
      unicode:            ($stdout.is_terminal and env.CLIUNICODE != null),
      verbose:            false,
    }
  );

def _opt_eval($rest):
  ( { argjson: (
        ( .argjson
        | if . then
            map(
              ( . as $a
              | .[1] |=
                try fromjson
                catch
                  ( "--argjson \($a[0]): \(.)"
                  | halt_error(_exit_code_args_error)
                  )
              )
            )
          end
        )
      ),
      color: (
        if .monochrome_output == true then false
        elif .color_output == true then true
        else null
        end
      ),
      expr: (
        # if -f was used, all rest non-args are filenames
        # otherwise first is expr rest is filesnames
        ( .expr_file
        | . as $expr_file
        | if . then
            try (open | tobytes | tostring)
            catch ("\($expr_file): \(.)" | halt_error(_exit_code_args_error))
          else $rest[0] // null
          end
        )
      ),
      expr_eval_path: .expr_file,
      filenames: (
        ( if .filenames then .filenames
          elif .expr_file then $rest
          else $rest[1:]
          end
        # null means stdin
        | if . == [] then [null] end
        )
      ),
      join_string: (
        if .join_output then ""
        elif .null_output then "\u0000"
        else null
        end
      ),
      null_input: (
        ( ( if .expr_file then $rest
            else $rest[1:]
            end
          ) as $files
        | if $files == [] and .repl then true
          else null
          end
        )
      ),
      raw_file: (
        ( .raw_file
        | if . then
            ( map(.[1] |=
                ( . as $f
                | try (open | tobytes | tostring)
                  catch ("\($f): \(.)" | halt_error(_exit_code_args_error))
                )
              )
            )
          end
        )
      ),
      raw_string: (
        if .raw_string
          or .join_output
          or .null_output
        then true
        else null
        end
      )
    }
  | with_entries(select(.value != null))
  );


def _opt_default_dynamic:
  ( stdout_tty as $stdout
  # TODO: intdiv 2 * 2 to get even number, nice or maybe not needed?
  | ( if $stdout.is_terminal then [_intdiv(_intdiv($stdout.width; 8); 2) * 2, 4] | max
      else 16
      end
    ) as $display_bytes
  | {
      display_bytes: $display_bytes,
      line_bytes: $display_bytes,
    }
  );

# these _to* function do a bit for fuzzy string to type conversions
def _opt_toboolean:
  try
    if . == "true" then true
    elif . == "false" then false
    else tonumber != 0
    end
  catch
    null;

def _opt_fromboolean: tostring;

def _opt_tonumber:
  try tonumber catch null;

def _opt_fromnumber: tostring;

def _opt_tostring:
  if . != null then
    ( "\"\(.)\""
    | try
        ( fromjson
        | if type != "string" then error end
        )
      catch null
    )
  end;

def _opt_fromstring: if . then tojson[1:-1] else "" end;

def _opt_toarray(f):
  try
    ( fromjson
    | if type == "array" and (all(f) | not) then null end
    )
  catch null;

def _opt_fromarray: tojson;

def _opt_is_string_pair:
  type == "array" and length == 2 and all(type == "string");

# TODO: cleanup
def _trim: capture("^\\s*(?<str>.*?)\\s*$"; "").str;

# "0-255=brightwhite,0=brightblack,32-126:9-13=white" -> [{"ranges": [[0-255]], value: "brightwhite"}, ...]
def _csv_ranges_to_array:
  ( split(",")
  | map(
    ( _trim
    | split("=")
    | { ranges:
          ( .[0]
          | split(":")
          | map(split("-") | map(tonumber))
          ),
        value: .[1]
      }
    ))
  );

def _opt_to_csv_ranges_array:
  try _csv_ranges_to_array
  catch null;

def _opt_from_csv_ranges_array:
  ( map(
      ( (.ranges | map(join("-")) | join(":"))
      + "="
      + .value
      )
    )
  | join(",")
  );

# "key=value,a=b,..." -> {"key": "value", "a": "b", ...}
def _csv_kv_to_obj:
  ( split(",")
  | map(_trim | split("=") | {key: .[0], value: .[1]})
  | from_entries
  );

def _opt_to_csv_kv_obj:
  try _csv_kv_to_obj
  catch null;

def _opt_from_csv_kv_obj:
  ( to_entries
  | map("\(.key)=\(.value)")
  | join(",")
  );

def _opt_cli_arg_tooptions:
  ( {
      addrbase:           (.addrbase | _opt_tonumber),
      arg:                (.arg | _opt_toarray(_opt_is_string_pair)),
      argjson:            (.argjson | _opt_toarray(_opt_is_string_pair)),
      array_truncate:     (.array_truncate | _opt_tonumber),
      bits_format:        (.bits_format | _opt_tostring),
      byte_colors:        (.byte_colors | _opt_to_csv_ranges_array),
      color:              (.color | _opt_toboolean),
      colors:             (.colors | _opt_to_csv_kv_obj),
      compact:            (.compact | _opt_toboolean),
      completion_timeout: (.array_truncate | _opt_tonumber),
      decode_file:        (.decode_file | _opt_toarray(_opt_is_string_pair)),
      decode_format:      (.decode_format | _opt_tostring),
      decode_progress:    (.decode_progress | _opt_toboolean),
      depth:              (.depth | _opt_tonumber),
      display_bytes:      (.display_bytes | _opt_tonumber),
      expr:               (.expr | _opt_tostring),
      expr_file:          (.expr_file | _opt_tostring),
      filenames:          (.filenames | _opt_toarray(type == "string")),
      force:              (.force | _opt_toboolean),
      include_path:       (.include_path | _opt_tostring),
      join_string:        (.join_string | _opt_tostring),
      line_bytes:         (.line_bytes | _opt_tonumber),
      null_input:         (.null_input | _opt_toboolean),
      raw_file:           (.raw_file| _opt_toarray(_opt_is_string_pair)),
      raw_output:         (.raw_output | _opt_toboolean),
      raw_string:         (.raw_string | _opt_toboolean),
      repl:               (.repl | _opt_toboolean),
      sizebase:           (.sizebase | _opt_tonumber),
      show_formats:       (.show_formats | _opt_toboolean),
      show_help:          (.show_help | _opt_toboolean),
      slurp:              (.slurp | _opt_toboolean),
      string_input:       (.string_input | _opt_toboolean),
      unicode:            (.unicode | _opt_toboolean),
      verbose:            (.verbose | _opt_toboolean),
    }
  | with_entries(select(.value != null))
  );

def _opt_cli_arg_fromoptions:
  ( {
      addrbase:           (.addrbase | _opt_fromnumber),
      arg:                (.arg | _opt_fromarray),
      argjson:            (.argjson | _opt_fromarray),
      array_truncate:     (.array_truncate | _opt_fromnumber),
      bits_format:        (.bits_format | _opt_fromstring),
      byte_colors:        (.byte_colors | _opt_from_csv_ranges_array),
      color:              (.color | _opt_fromboolean),
      colors:             (.colors | _opt_from_csv_kv_obj),
      compact:            (.compact | _opt_fromboolean),
      completion_timeout: (.array_truncate | _opt_fromnumber),
      decode_file:        (.decode_file | _opt_fromarray),
      decode_format:      (.decode_format | _opt_fromstring),
      decode_progress:    (.decode_progress | _opt_fromboolean),
      depth:              (.depth | _opt_fromnumber),
      display_bytes:      (.display_bytes | _opt_fromnumber),
      expr:               (.expr | _opt_fromstring),
      expr_file:          (.expr_file | _opt_fromstring),
      filenames:          (.filenames | _opt_fromarray),
      force:              (.force | _opt_fromboolean),
      include_path:       (.include_path | _opt_fromstring),
      join_string:        (.join_string | _opt_fromstring),
      line_bytes:         (.line_bytes | _opt_fromnumber),
      null_input:         (.null_input | _opt_fromboolean),
      raw_file:           (.raw_file| _opt_fromarray),
      raw_output:         (.raw_output | _opt_fromboolean),
      raw_string:         (.raw_string | _opt_fromboolean),
      repl:               (.repl | _opt_fromboolean),
      sizebase:           (.sizebase | _opt_fromnumber),
      show_formats:       (.show_formats | _opt_fromboolean),
      show_help:          (.show_help | _opt_fromboolean),
      slurp:              (.slurp | _opt_fromboolean),
      string_input:       (.string_input | _opt_fromboolean),
      unicode:            (.unicode | _opt_fromboolean),
      verbose:            (.verbose | _opt_fromboolean),
    }
  | with_entries(select(.value != null))
  );

def _opt_cli_opts:
  {
    "arg": {
      long: "--arg",
      description: "Set variable $NAME to string VALUE",
      pairs: "NAME VALUE"
    },
    "argjson": {
      long: "--argjson",
      description: "Set variable $NAME to JSON",
      pairs: "NAME JSON"
    },
    "compact": {
      short: "-c",
      long: "--compact-output",
      description: "Compact output",
      bool: true
    },
    "color_output": {
      short: "-C",
      long: "--color-output",
      description: "Force color output",
      bool: true
    },
    "decode_format": {
      short: "-d",
      long: "--decode",
      description: "Decode format (probe)",
      string: "NAME"
    },
    "decode_file": {
      long: "--decode-file",
      description: "Set variable $NAME to decode of file",
      pairs: "NAME PATH"
    },
    "expr_file": {
      short: "-f",
      long: "--from-file",
      description: "Read EXPR from file",
      string: "PATH"
    },
    "show_help": {
      short: "-h",
      long: "--help",
      description: "Show help for TOPIC (ex: --help, --help formats)",
      string: "[TOPIC]",
      optional: true
    },
    "join_output": {
      short: "-j",
      long: "--join-output",
      description: "No newline between outputs",
      bool: true
    },
    "include_path": {
      short: "-L",
      long: "--include-path",
      description: "Include search path",
      array: "PATH"
    },
    "null_output": {
      short: "-0",
      long: "--null-output",
      # for jq compatibility
      aliases: ["--nul-output"],
      description: "Null byte between outputs",
      bool: true
    },
    "null_input": {
      short: "-n",
      long: "--null-input",
      description: "Null input (use input and inputs functions to read)",
      bool: true
    },
    "monochrome_output": {
      short: "-M",
      long: "--monochrome-output",
      description: "Force monochrome output",
      bool: true
    },
    "option": {
      short: "-o",
      long: "--option",
      description: "Set option (ex: -o color=true, see --help options)",
      object: "KEY=VALUE",
    },
    "string_input": {
      short: "-R",
      long: "--raw-input",
      description: "Read raw input strings (don't decode)",
      bool: true
    },
    "raw_file": {
      long: "--raw-file",
      # for jq compatibility
      aliases: ["--raw-file"],
      description: "Set variable $NAME to string content of file",
      pairs: "NAME PATH"
    },
    "raw_string": {
      short: "-r",
      # for jq compat, is called raw string internally, "raw output" is if
      # we can output raw bytes or not
      long: "--raw-output",
      description: "Raw string output (without quotes)",
      bool: true
    },
    "repl": {
      short: "-i",
      long: "--repl",
      description: "Interactive REPL",
      bool: true
    },
    "slurp": {
      short: "-s",
      long: "--slurp",
      description: "Read (slurp) all inputs into an array",
      bool: true
    },
    "show_version": {
      short: "-v",
      long: "--version",
      description: "Show version",
      bool: true
    },
  };

def options($opts):
  [_opt_default_dynamic] + _options_stack + [$opts] | add;
def options: options({});
