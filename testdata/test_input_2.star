"""example to render multiline items"""

def foo(a, b, c = 4):
    source = {
        "z": 12,
        "c": 14,
        "d": {
            "foo": 12,
            "bar": {
                "foobar": "test",
                "sub_bar": 2,
            },
        },
        "lastone": {"foo": "bar",},
    }
    t1 = (1,)
    t2 = (1, 2, 3,)
    call_something(
        3,
        15,
        foobar,
        [
            a,
            12,
            (3, 4, 2,),
            {},
        ],
        {
            "a": "b",
            "bar": {
                "foobar": "test",
                "sub_bar": 2,
            },
        },
    )
    pass
