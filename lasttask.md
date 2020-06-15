1.  If a conditional execution step expression is not evaluated to be true, the compiler treats the conditional as a hardcoded "false" which makes it disregard any code
    that the conditional may point to.

    [SOLUTION]
    Only reject actual hardcoded falses and maybe generate code that evaluates the expression at runtime.
    "Reject Known Bad" instead of "Accept Known Good"

    [STATUS]
    FIXED

2.  Currently, variables within functions are not locally scoped (YIKES!!). The "Method" struct needs to be able to contain locally scoped variables.

    [POSSIBLE SOLUTIONS]

    1.  Maybe make the "Method" struct inherit the "AST" struct, this way anything declared within it will be treated as a private declaration unless specified
        otherwise.

        PROS:
        -   Easy to implement.
        -   Variables expect their scopes to be "AST" struct pointers and it will be hard to make them recognize "Method" structs as alternatives.

        CONS:
        -   The "AST" struct implements a lot of functionality that methods don't need e.g package scoping. This will lead to wastage of resources.
    
    2.  Re-implement variable declaration by adding a "Variables" field to the "Method" struct and add variable declaration logic to "CompileEntries()".

        PROS:
        -   Fixes con 1 in above solution.

        CONS:
        -   Needs more effort and thought.
    
    [STATUS]
    FIXED: Using solution 2

3. "false" not being captured by tokenizer due to participle template declaration error

    [SOLUTION]
    Change template typ from *bool to string and decide whether it is true or false in evaluate.go

    [STATUS]
    FIXED

4. Imported functions not being built in final C source