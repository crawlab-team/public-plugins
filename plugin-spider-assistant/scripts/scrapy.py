import ast
import argparse
import json
import os
import configparser

parser = argparse.ArgumentParser()
parser.add_argument('-a', dest='action', default='all')
parser.add_argument('-d', '--dir', dest='directory', default='.')
parser.add_argument('-f', '--file', dest='filepath')
args = parser.parse_args()

_middleware_methods = [
    'from_crawler',
    'process_spider_input',
    'process_spider_output',
    'process_spider_exception',
    'process_start_requests',
    'spider_opened',
]


def parse_ast(filepath: str) -> ast.Module:
    # ast object
    ast_obj = _parse_ast(filepath)
    print(ast.dump(ast_obj, indent=4))
    return ast_obj


def parse_all() -> dict:
    result = {}

    # settings
    res_settings = parse_settings()
    if res_settings is not None:
        result['settings'] = res_settings

    # items
    res_items = parse_items()
    if res_items is not None:
        result['items'] = res_items

    # spiders
    res_spiders = parse_spiders()
    if res_spiders is not None:
        result['spiders'] = res_spiders

    # middlewares
    res_middlewares = parse_middlewares()
    if res_middlewares is not None:
        result['middlewares'] = res_middlewares

    # scrapy.cfg
    res_cfg = parse_scrapy_cfg()
    if res_cfg is not None:
        result['cfg'] = res_cfg

    return result


def parse_scrapy_cfg() -> [dict, None]:
    cfg = _get_scrapy_cfg()
    if cfg is None:
        return
    res = {}
    for section in cfg.values():
        res[section.name] = {}
        for option_name, option in section.items():
            res[section.name][option_name] = option
    return res


def parse_items(filepath: str = None) -> list:
    # default file path if empty
    if filepath is None:
        module_name = _get_default_module()
        filepath = os.path.join(args.directory, module_name, 'items.py')

    # ast object
    ast_obj = _parse_ast(filepath)

    # results
    results = []

    # iterate body elements
    for el in ast_obj.body:
        # skip if type is not ast.ClassDef
        if type(el) != ast.ClassDef:
            continue

        # parsed results
        res = _parse_items_element(el)

        # file path
        res['filepath'] = filepath.replace(args.directory, '')

        # skip if result is empty
        if res is None:
            continue

        # add to results
        results.append(res)

    return results


def parse_settings(filepath: str = None) -> list:
    # default file path if empty
    if filepath is None:
        module_name = _get_default_module()
        filepath = os.path.join(args.directory, module_name, 'settings.py')

    # ast object
    ast_obj = _parse_ast(filepath)

    # results
    results = []

    # iterate body elements
    for el in ast_obj.body:
        # skip if type is not ast.Assign
        if type(el) != ast.Assign:
            continue

        # parsed result
        res = _parse_settings_element(el)

        # file path
        res['filepath'] = filepath.replace(args.directory, '')

        # skip if result is empty
        if res is None:
            continue

        # add to results
        results.append(res)

    return results


def parse_spiders(dirpath: str = None) -> list:
    # default directory path if empty
    if dirpath is None:
        module_name = _get_default_module()
        dirpath = os.path.join(args.directory, module_name, 'spiders')

    # results
    results = []

    for filename in os.listdir(dirpath):
        # skip if not .py
        if not filename.endswith('.py'):
            continue

        # file path
        filepath = os.path.join(dirpath, filename)

        # parsed result
        _results = _parse_spider_file(filepath)

        # add to results
        for res in _results:
            results.append(res)

    return results


def parse_middlewares(filepath: str = None) -> list:
    # default file path if empty
    if filepath is None:
        module_name = _get_default_module()
        filepath = os.path.join(args.directory, module_name, 'middlewares.py')

    # ast object
    ast_obj = _parse_ast(filepath)

    # results
    results = []

    # iterate body elements
    for el in ast_obj.body:
        # skip if type is not ast.ClassDef
        if type(el) != ast.ClassDef:
            continue

        # parsed results
        res = _parse_middlewares_elements(el)

        # file path
        res['filepath'] = filepath.replace(args.directory, '')

        # skip if result is empty
        if res is None:
            continue

        # add to results
        results.append(res)

    return results


def _parse_items_element(el) -> [dict, None]:
    # result
    res = {}

    # skip if bases is empty
    if len(el.bases) == 0:
        return

    # base
    b = el.bases[0]

    # skip if base is not valid
    if not _is_type(b, 'scrapy', 'Item'):
        return

    # item name
    res['name'] = el.name

    # item fields
    res['fields'] = []

    # iterate sub statements
    for stmt in el.body:
        # skip if type is not valid
        if type(stmt) != ast.Assign or type(stmt.value) != ast.Call:
            continue

        # skip if func is not valid
        if not _is_type(stmt.value.func, 'scrapy', 'Field'):
            continue

        # skip if targets empty
        if len(stmt.targets) == 0:
            continue

        # target
        tgt: ast.Name = stmt.targets[0]

        # skip if target is not ast.Name
        if type(tgt) != ast.Name:
            continue

        # field name
        field_name = tgt.id

        # field type
        field_type = ''
        if len(stmt.value.keywords) > 0:
            for k in stmt.value.keywords:
                if type(k) != ast.keyword:
                    continue
                if k.arg != 'serializer':
                    continue
                if type(k.value) != ast.Name:
                    continue
                # field type
                field_type = k.value.id
                break

        # field
        field = {
            'name': field_name,
            'type': field_type,
        }

        # add to fields
        res['fields'].append(field)

    return res


def _parse_settings_element(el) -> [dict, None]:
    # result
    res = {}

    # skip if targets is empty
    if len(el.targets) == 0:
        return

    # target
    tgt: ast.Name = el.targets[0]

    # skip if type is not ast.Name
    if type(tgt) != ast.Name:
        return

    # setting name
    res['name'] = tgt.id

    # element value
    ev = el.value

    # value type
    vt = type(ev)

    # if value is ast.Constant
    if vt == ast.Constant:
        _res = _get_constant_result(ev)
        res['type'] = _res['type']
        res['value'] = _res['value']

    # if value is ast.List
    if vt == ast.List:
        res['type'] = 'list'
        res['value'] = []
        # iterate sub elements
        for sub_el in ev.elts:
            # skip if sub element is not ast.Constant
            if type(sub_el) != ast.Constant:
                return
            res['value'].append(_get_constant_result(sub_el))

    # if value is ast.Dict
    if vt == ast.Dict:
        res['type'] = 'dict'
        res['value'] = []
        # iterate keys and values
        for i in range(len(ev.values)):
            key = ev.keys[i]
            value = ev.values[i]
            # skip if key or value is not ast.Constant
            if type(key) != ast.Constant or type(value) != ast.Constant:
                return
            res['value'].append({
                'key': _get_constant_result(key),
                'value': _get_constant_result(value),
            })

    return res


def _parse_spider_file(filepath: str) -> list:
    # ast object
    ast_obj = _parse_ast(filepath)

    # results
    results = []

    # iterate body elements
    for el in ast_obj.body:
        # skip if type is not ast.ClassDef
        if type(el) != ast.ClassDef:
            continue

        # parsed result
        res = _parse_spider(el)

        # file path
        res['filepath'] = filepath.replace(args.directory, '')

        # skip if result is empty
        if res is None:
            continue

        # add to results
        results.append(res)

    return results


def _parse_spider(el: ast.ClassDef) -> [dict, None]:
    # class name
    class_name = _get_class_name(el)

    # skip if class name not ends with 'Spider'
    if not class_name.endswith('Spider'):
        return

    # result
    res = {
        'name': el.name,
        'type': class_name,
    }

    return res


def _parse_middlewares_elements(el: ast.ClassDef) -> [dict, None]:
    # middleware
    middleware = {
        'name': el.name,
    }

    # methods
    methods = []

    for sub_el in el.body:
        # skip if type is not ast.FunctionDef
        if type(sub_el) is not ast.FunctionDef:
            continue

        # method name
        method_name = sub_el.name

        # add to methods if in standard method list
        if method_name in _middleware_methods:
            methods.append(method_name)

    # return None if not match
    if len(methods) == 0:
        return

    # methods
    middleware['methods'] = methods

    return middleware


def _get_constant_result(el: ast.Constant) -> dict:
    return {
        'type': type(el.value).__name__,
        'value': el.value,
    }


def _parse_ast(filepath: str) -> ast.Module:
    with open(filepath) as f:
        src = f.read()

    res = ast.parse(src)

    return res


def _is_type(el, m: str, t: str) -> bool:
    if type(el) == ast.Name:
        if el.id != t:
            return False
    elif type(el) == ast.Attribute:
        if type(el.value) != ast.Name or el.value.id != m:
            return False
        if el.attr != t:
            return False
    else:
        return False
    return True


def _get_class_name(el: ast.ClassDef) -> [str, None]:
    if len(el.bases) == 0:
        return

    b = el.bases[0]

    if type(b) == ast.Name:
        return b.id
    elif type(b) == ast.Attribute:
        if type(b.value) != ast.Name:
            return
        return b.attr
    else:
        return


def _get_scrapy_cfg() -> [configparser.ConfigParser, None]:
    cfg_path = os.path.join(args.directory, 'scrapy.cfg')
    if not os.path.exists(cfg_path):
        return
    config = configparser.ConfigParser()
    config.read(cfg_path)
    return config


def _get_default_module() -> [str, None]:
    config = _get_scrapy_cfg()
    if config is None:
        return
    try:
        settings = config.get('settings', 'default')
        arr = settings.split('.')
        if len(arr) == 0:
            return None
        return arr[0]
    except configparser.NoSectionError:
        return
    except configparser.NoOptionError:
        return


def main():
    results = None
    if args.action == 'ast':
        parse_ast(args.filepath)
        return

    if args.action == 'spiders':
        results = parse_spiders()

    if args.action == 'items':
        results = parse_items(args.filepath)

    if args.action == 'settings':
        results = parse_settings(args.filepath)

    if args.action == 'middlewares':
        results = parse_middlewares(args.filepath)

    if args.action == 'all':
        results = parse_all()

    # output
    print(json.dumps(results))


if __name__ == '__main__':
    main()
