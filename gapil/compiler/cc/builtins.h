#include <stdint.h>
#include <stddef.h>
typedef struct pool_t    pool;
typedef struct globals_t globals;
typedef struct string_t  string;

static const uint32_t ERR_SUCCESS = 0;
static const uint32_t ERR_ABORTED = 1;

static const uint64_t mapElementEmpty = 0;
static const uint64_t mapElementFull = 1;
static const uint64_t mapElementUsed = 2;

static const uint64_t mapGrowMultiplier = 2;
static const uint64_t minMapSize = 16;
static const float mapMaxCapacity = 0.8f;


typedef struct context_t {
	uint32_t    id;
	uint32_t    location;
	globals*    globals;
	pool*       app_pool;
	string*     empty_string;
} context;

typedef struct pool_t {
	uint32_t ref_count;
	void*    buffer;
} pool;

typedef struct slice_t {
	pool*    pool; // The underlying pool.
	void*    root; // Original pointer this slice derives from.
	void*    base; // Address of first element.
	uint64_t size; // Size in bytes of the slice.
} slice;

typedef struct string_t {
	uint32_t ref_count;
	uint64_t length;
	uint8_t  data[1];
} string;

typedef struct map_t {
	uint32_t ref_count;
	uint64_t count;
	uint64_t capacity;
	void*    elements;
} map;

#ifndef DECL_GAPIL_CALLBACK
#define DECL_GAPIL_CALLBACK(RETURN, NAME, ...) RETURN NAME(__VA_ARGS__)
#endif

DECL_GAPIL_CALLBACK(void*,   gapil_alloc,             context* ctx, uint64_t size, uint64_t align);
DECL_GAPIL_CALLBACK(void*,   gapil_realloc,           context* ctx, void* ptr, uint64_t size, uint64_t align);
DECL_GAPIL_CALLBACK(void,    gapil_free,              context* ctx, void* ptr);
DECL_GAPIL_CALLBACK(void,    gapil_apply_reads,       context* ctx);
DECL_GAPIL_CALLBACK(void,    gapil_apply_writes,      context* ctx);
DECL_GAPIL_CALLBACK(void,    gapil_free_pool,         context* ctx, pool*);
DECL_GAPIL_CALLBACK(void,    gapil_make_slice,        context* ctx, uint64_t size, slice* out);
DECL_GAPIL_CALLBACK(void,    gapil_copy_slice,        context* ctx, slice* dst, slice* src);
DECL_GAPIL_CALLBACK(void,    gapil_pointer_to_slice,  context* ctx, uint64_t ptr, uint64_t offset, uint64_t size, slice* out);
DECL_GAPIL_CALLBACK(string*, gapil_pointer_to_string, context* ctx, uint64_t ptr);
DECL_GAPIL_CALLBACK(string*, gapil_slice_to_string,   context* ctx, slice* slice);
DECL_GAPIL_CALLBACK(string*, gapil_make_string,       context* ctx, uint64_t length, void* data);
DECL_GAPIL_CALLBACK(void,    gapil_free_string,       context* ctx, string* string);
DECL_GAPIL_CALLBACK(void,    gapil_string_to_slice,   context* ctx, string* string, slice* out);
DECL_GAPIL_CALLBACK(string*, gapil_string_concat,     context* ctx, string* a, string* b);
DECL_GAPIL_CALLBACK(int32_t, gapil_string_compare,    context* ctx, string* a, string* b);
DECL_GAPIL_CALLBACK(void,    gapil_call_extern,       context* ctx, string* name, void* args, void* res);
DECL_GAPIL_CALLBACK(void,    gapil_logf,              context* ctx, uint8_t severity, uint8_t* fmt, ...);

#undef DECL_GAPIL_CALLBACK