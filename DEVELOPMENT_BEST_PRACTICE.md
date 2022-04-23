# Development Best Practice

Software development is like nature: it's in a constant state of flux and continuously changing. As software engineers, it's our responsibility to embrace the flux and write clean code that is _optimized for change_. In doing so, we can rapidly innovate and satisfy customer needs.

## Guiding principles

These principles provide a guide for designing software. At times they may seem contradictory, but they ultimately aim to produce the same kind of code: _simple_ and _maintainable_.

> "Debugging is twice as hard as writing the code in the first place. Therefore, if you write the code as cleverly as possible, you are, by definition, not smart enough to debug it." - _Brian W. Kernighan_

- [SOLID](https://www.digitalocean.com/community/conceptual_articles/s-o-l-i-d-the-first-five-principles-of-object-oriented-design#single-responsibility-principle) - The Single-responsibility Principle, Open-closed Principle, Liskov Substitution Principle, Interface Segregation Principle and Depdendency Inversion Principle collectively define a guide for writing maintainable code. While often referenced in the context of an object-oriented language, they are applicable to all languages.
- [LoD](https://en.wikipedia.org/wiki/Law_of_Demeter) - The Law of Demeter tells us a construct should talk to their direct dependencies and _only_ their direct dependencies. Reaching to transitive dependencies creates complex layers of interaction that drive toward [spaghetti code](https://en.wikipedia.org/wiki/Spaghetti_code).
- [KISS](https://people.apache.org/~fhanik/kiss.html) - "Keep it simple, stupid" was coined by the US Navy. Systems should be designed as simply as possible.
- [YAGNI](https://martinfowler.com/bliki/Yagni.html) - "You aren't gonna need it" if you don't have a concrete use-case, so don't write it.
- [DRY](https://en.wikipedia.org/wiki/Don%27t_repeat_yourself) - "Don't repeat yourself" suggests you should preference code re-use over duplication. However, you should [avoid hasty abstractions](https://sandimetz.com/blog/2016/1/20/the-wrong-abstraction) as the wrong abstraction can be extremely costly to correct. Instead, lean into small amounts of duplication to help identify the right abstractions through multiple use cases and define the pathway to DRY design.

## "clean code that is optimized for change"

What constitutes clean code? We want simple and maintable, but how do we define that? The following best practices offer niche advice. Combined, they represent our definition of clean code.

### State

> Shared mutable state is believed by many to be the “root of all evil”, or at least the cause of most of the accidental complexity in our code. And “Complexity is the root cause of the vast majority of problems with software today.” - [Mauro Bieg](https://mb21.github.io/blog/2021/01/23/pure-functional-programming-and-shared-mutable-state.html#the-root-of-all-evil)

Package level state is rarely required. If it is, it should be composed of re-usable constructs defined and exposed from the package itself.

In avoiding package level state we avoid [hidden dependencies and unintended side effects](https://dave.cheney.net/practical-go/presentations/gophercon-israel.html#_avoid_package_level_state) stemming from global state mutation. Model loosely coupled components by [declaring your dependencies](#dependencies).

### Names

##### Packages

Package names help form the first impression of what a package provides and is used to reference the public API of the package. A good package name brings about clarity and purpose for consumers. Package names should be concise and represent the behavior the package provides not the types it contains. For example, `meals` provides `dr

Avoid stutter in struct and interface names by treating the package name as a namespace and considering how the types are consumed. For example, `drink.DrinkDecorator` should be `drink.Dectorator`.

If you find yourself using an appropriate package name that's commonly used as a variable, consider pluralising the package name.

Avoid (like the plague) `types`, `interfaces`, `common`, `util` or `base` packages. They don't represent anything cohesive and tend to collect unrelated types and algorithms that often require teething apart as a project progresses due to cyclic dependencies. If a utility style package is required, make it specific. For example a `/utils/cmdutil` package may provide command utility functions.

##### Function & method names

Functions and methods should adequetly describe their behavior. Generally, they should follow a verb-noun form.

##### Variable names

Variable names should be concise and descriptive. Prefer single word names. The further away from the site of declaration a variable is used, the more descriptive it needs to be. For example, a variable named `n` used 30 lines after declaration makes it unnecessarily difficult to reason what it represents at the site of use.

### Abstractions

- Functions & Methods
- Constructs
- Interfaces

### <a nane="dependencies"></a> Depencencies

- Accept interfaces, return structs
- Declare your dependencies

### Package API

- this might be wrapped up in naming and testing. Do we have more explicit points?

### Testing

- test using the `_test` idiom
- Use test to help identify if the package is easy to use for consumers
- isolating concerns like io

### Interfaces

- Define the behavior a _consumer_ expects
- Premature interfaces
- Accept interfaces
- Should be cohesive
- Should be small
- Preferably single method

### Errors

- Describe the action that failed
- Use lower case
- Don't prepend error
- Don't overshare information 
- Add context if you can

### Boolean logic

- Toggling behavior using boolean values is generally bad
- Create dynamic constructs that can be built without the behavior

### Concurrency

- Prefer synchonous APIs
- Leave it to the caller
- Mask from the caller

### Disambiguating context.Context

- Represents the context of execution
    - Useful for tear down
- There should be 1 and only 1 context
- Should be the first parameter on the primary call path

### Loggers

- They're just another dependency, inject them.

###  Comments

- All public APis should be documented
- May seem redundant at times, consider documentation from a `godoc` point of view.
- Packages _should_ contain package level documentation
- What makes a good comment?

### Types

- I forgot what this was for...

### Variable initialization

- What forms to preference under what circumstances

### Channels

- If your API leverages them, inject them
- Document channel behavior

### Returns

- Return early
- Naked returns should be avoided

### Panicing

- General advice is to not panic
- Exceptions and rationale around invalidity of a program


## References

- Dave Cheney - https://dave.cheney.net/practical-go/presentations/gophercon-israel.html
- Peter Bourgon - https://peter.bourgon.org/go-best-practices-2016/
- Effective Go - https://go.dev/doc/effective_go
- Code Review Comments - https://github.com/golang/go/wiki/CodeReviewComments