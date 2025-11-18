package eino

const mindMapSchemaString = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "mapId": {"type": "string"},
    "userId": {"type": "string"},
    "title": {"type": "string"},
    "desc": {"type": "string"},
    "layout": {"type": "string"},
    "root": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "data": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "text": {"type": "string"},
            "expand": {"type": "boolean"},
            "isActive": {"type": "boolean"},
            "uid": {"type": "string"}
          },
          "required": ["text", "uid"]
        },
        "children": {
          "type": "array",
          "items": {"$ref": "#/$defs/node"}
        },
        "smmVersion": {"type": "string"}
      },
      "required": ["data", "children"]
    },
    "createdAt": {"type": "string"},
    "updatedAt": {"type": "string"}
  },
  "required": ["mapId", "userId", "title", "root", "createdAt", "updatedAt"],
  "$defs": {
    "node": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "data": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "text": {"type": "string"},
            "expand": {"type": "boolean"},
            "isActive": {"type": "boolean"},
            "uid": {"type": "string"}
          },
          "required": ["text", "uid"]
        },
        "children": {
          "type": "array",
          "items": {"$ref": "#/$defs/node"}
        }
      },
      "required": ["data", "children"]
    }
  }
}`

const generateMindMapSchemaString = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "mapId": {"type": "string", "enum": ["xxx"]},
    "userId": {"type": "string", "description":"用户唯一id，聊天记录中会给出"},
    "title": {
      "type": "string",
      "description": "标题，不超过15个字，概括文本核心主题，内部禁止包含换行符或\\n"
    },
    "desc": {
      "type": "string",
      "description": "描述，不超过30个字，简要说明导图内容，内部禁止包含换行符或\\n"
    },
    "layout": {"type": "string", "enum": ["mindMap"]},
    "root": {"$ref": "#/$defs/node"}
  },
  "required": ["mapId", "userId", "title", "desc", "layout", "root"],
  "$defs": {
    "node": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "data": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "text": {
              "type": "string",
              "description": "节点文本，8-20个字，突出关键词，内部禁止包含换行符或\\n"
            }
          },
          "required": ["text"]
        },
        "children": {
          "type": "array",
          "items": {"$ref": "#/$defs/node"},
          "description": "子节点数组，叶子节点为空数组[]"
        }
      },
      "required": ["data", "children"]
    }
  }
}`
