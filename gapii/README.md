# Graphics API Interceptor (GAPII)

The Graphics API Interceptor is the AGI component responsible for intercepting and capturing all commands issued by an application to the graphics driver.

GAPII can be packaged as a dynamically-loaded library (`.so`, `.dll`) or as an archive linked into the application at build time (`.a`, `.lib`).

GAPII provides a full set of functions that entirely replaces the target graphics API to be intercepted. These GAPII provided functions are used by the application instead of the regular graphics driver.

GAPII behaves as follows:

* On the first call to an intercepted function, GAPII will block, listening for a socket connection.
* Once a connection has been made, GAPII will allow the application to continue execution.
* If GAPII does not capture the trace from the beginning, it will monitor the graphics state instead.
* For each intercepted function that’s called by the application, the function’s identifier and arguments are encoded and streamed to the socket.
* Memory observations are also streamed for functions that read from or write to memory.
* If the remote endpoint requests to end the capture, or the specified amount of frames have been captured or the application is terminated, an end-of-stream marker is written.
* The remote endpoint of the socket closes the socket after receiving the end-of-stream marker.

## State tracking

OpenGL ES is a complicated API that has some commands that are tricky to capture.

Take for example `glVertexAttribPointer` which is used to assign data to a vertex attribute:

```
void glVertexAttribPointer(GLuint         index,
                           GLint          size,
                           GLenum         type,
                           GLboolean      normalized,
                           GLsizei        stride,
                           const GLvoid * pointer);
```

The last parameter can be interpreted as a client-side pointer to the vertex data, or a byte offset into a server-side buffer depending on whether there is a currently bound buffer to `GL_ARRAY_BUFFER_BINDING`. GAPII needs to differentiate between these two different interpretations of the pointer argument as the data at pointer (if it is indeed a pointer) will need to be observed in order to fully capture the trace.

Understanding whether the binding is active or not requires knowledge of the driver state. This can be queried but this might cause unnecessary pipeline stalls in the driver, and GAPII should make as few artificial calls to the driver as possible, in order to avoid triggering unintentional side-effects.

In order to solve these issues, GAPII contains a state-mutator code-generated from the graphics API file, similar to the state-mutator logic in GAPIS.


## Calling the real driver function

Each intercepted function needs to call the real driver function and also perform state-mutator logic based on the parameters (and sometimes the return value) of the call. Because the state-mutator is imitating the driver to track driver state based on inputs and outputs of the call, the state-mutation logic needs to be split into pre-call and post-call statements. The splitting point between the pre and post statements is called a **fence**, and this is the point where the call to the real driver is made.

Pre-call statements may include memory observations on the pointer arguments handed to the function. Post-call statements may include logic dependent on the return value of the call. More details about fences can be [found here](../gapil/README.md#fence).


## Thread handling

As many graphics APIs support concurrent usage, GAPII, being a wrapper over the driver, also needs to support concurrent calls.

Many graphics APIs have the concept of contexts that can be created and destroyed.

Some graphics APIs allow contexts to be handed between threads by transferring ownership.

Some graphics APIs permit simultaneous sharing of contexts by multiple threads.

GAPII needs to support all of these cases.

Each encoded command holds an identifier of the thread that made the call.
Commands from all threads are encoded into a single chronological stream.