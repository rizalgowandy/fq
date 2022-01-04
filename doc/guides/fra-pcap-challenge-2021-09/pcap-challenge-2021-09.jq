#!/usr/bin/env fq -rf

# ["tag"] -> <tag/>
# ["tag", {k: v, ...}] -> <tag k="v" .../>
# ["tag", {k: v, ...}, ["tag", ...]] -> <tag k="v" ...><tag/> ...</tag>
def toxml($indent; $nl):
  def _f($d):
    def _attr: to_entries | map("\(.key)=\(.value | tostring | tojson)") | join(" ");
    if .[2] then
      ( "\($d)<\(.[0]) \(.[1] | _attr)>"
      , (.[2][] | _f($d+$indent))
      , "\($d)</\(.[0])>"
      )
    elif .[1] then "\($d)<\(.[0]) \(.[1] | _attr)/>"
    else "\($d)<\(.[0])/>"
    end;
  ( [_f("")]
  | join($nl)
  );
def toxml($indent): toxml($indent; "\n");
def toxml: toxml("  ");

# xor input array with key array
# key will be repeated to fit length of input
def xor_array($key):
  # [1,2,3] | repeat(7) -> [1,2,3,1,2,3,1]
  def repeat($len):
    ( length as $l
    | [.[range($len) % $l]]
    );
  ( . as $input
  # [$input, $key repeated]
  | [ $input
    , ($key | repeat($input | length))
    ]
  # [[$input[0], $key[0], ...]
  | transpose
  | map(bxor(.[0]; .[1]))
  );

( first(.uncompressed.files[] | select(.name == "triangle.pcap"))
| .data.tcp_connections[0].server_stream
| split("\n")
| map(
  ( select(. != "")
  | fromjson
  | if .encoding == "xor" then
      .data |= (hex | explode | xor_array([71])| implode)
    end
  | if .encoding == "long_xor" then
      ( .data |=
        ( hex
        | explode
        | xor_array("GravityForce" | explode)
        | implode
        )
      )
    end
  | if .msg_type == "update" then .data |= fromjson end
  )) as $msgs
| [ "svg",
    # move viewbox to where the objects are
    { viewBox: "50 120 350 350",
      width: 350,
      height: 350,
      xmlns: "http://www.w3.org/2000/svg"
    },
    [ # black background
      ["rect", {fill: "#101010", x: 50, y: 120, width: 350, height: 350}]
      # draw blue dotted line between player moves
    , [ "polyline"
      , { fill: "none"
        , stroke: "#5050d0"
        , "stroke-dasharray": "5 10"
        , points:
          ( [ $msgs[]
            | select(.msg_type == "update" and .data.type == "Player")
            | .data.x, .data.y
            ]
          | join(" ")
          )
        }
      ]
      # gather last update for all objects and draw them
    , ( reduce ($msgs[] | select(.msg_type == "update")) as $msg (
          {};
          # use tostring as object keys can only be strings
          .[$msg.data.id | tostring] = $msg.data
        )
      | { Player: {style: "fill: #0000ff", size: 15},
          Flower: {style: "fill: #00d000", size: 10},
          Rock: {style: "fill: #a0a0a0", size: 8}
        } as $types
      | .[]
      | $types[.type] as $t
      | [ "rect",
          { width: $t.size,
            height: $t.size,
            style: $t.style,
            transform: "rotate(\(.rot) \(.x-$t.size/2) \(.y-$t.size/2))",
            x: (.x-$t.size/2),
            y: (.y-$t.size/2)
          }
        ]
      )
    ]
  ]
| toxml
)
