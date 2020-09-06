"""example starlark file
slightly modified example from go.starklark.net
Some quotes just in case \"\"\"
"""

load("import.star", "foo", test = "bar")

number = 18

20

string_value, string_value_2 = "A", "C"

people = {"Alice": 22, "Bob": 40, "Charlie": 55, "Dave": 14}

names = ", ".join(people.keys())

mytuple = (1, 2, 3)

def greet(name):
    """Return a greeting."""
    return "Hello {}!".format(name)

greeting = greet(names)

above30 = [name for name, age in people.items() if age >= 30]

print("{} people are above 30.".format(len(above30)))

def fibonacci(n):
    res = list(range(n))
    for i in res[2:]:
        res[i] = res[i - 2] + res[i - 1]
    return res

def custom_args(*args, **kwargs):
    print(args)
    print(kwargs)
    if "n" in kwargs:
        fizz_buzz(**kwargs)

def is_default(a, b, c = 10):
    return c == 10

def check_default():
    return is_default(b = 10, a = 20, c = 30)

def fizz_buzz(n):
    """Function description.

    Args:
        n: the last number in sequence,
        should be equal or greater than 1.
    """
    for i in range(1, n + 1):
        s = ""
        if i % 3 == 0:
            s += "Fizz"
        if i % 5 == 0:
            s += "Buzz"
        print(s if s else i)

fizz_buzz(20)

fibonacci(200)
