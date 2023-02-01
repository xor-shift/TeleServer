#include <cstdint>
#include <limits>
#include <random>
#include <span>

#include <fmt/format.h>

static std::random_device s_rd{};

template<typename T, typename U = T>
static constexpr T rotl(T x, U k_arg) {
    const auto k = static_cast<T>(k_arg);
	return (x << k) | (x >> (std::numeric_limits<T>::digits - k));
}

// https://prng.di.unimi.it/xoroshiro64star.c
uint32_t xoroshiro64s_next(uint32_t (&s)[2]) {
	const uint32_t s0 = s[0];
	uint32_t s1 = s[1];
	const uint32_t result = s0 * 0x9E3779BB;

	s1 ^= s0;
	s[0] = rotl(s0, 26) ^ s1 ^ (s1 << 9); // a, b
	s[1] = rotl(s1, 13); // c

	return result;
}

// https://prng.di.unimi.it/xoroshiro128plusplus.c
uint64_t xoroshiro128pp_next(uint64_t (&s)[2]) {
	const uint64_t s0 = s[0];
	uint64_t s1 = s[1];
	const uint64_t result = rotl(s0 + s1, 17) + s0;

	s1 ^= s0;
	s[0] = rotl(s0, 49) ^ s1 ^ (s1 << 21); // a, b
	s[1] = rotl(s1, 28); // c

	return result;
}

void xoroshiro128pp_jump(uint64_t (&s)[2]) {
	static const uint64_t JUMP[] = { 0x2bd7a6a6e99c2ddc, 0x0992ccaf6a6fca05 };

	uint64_t s0 = 0;
	uint64_t s1 = 0;
	for(int i = 0; i < sizeof JUMP / sizeof *JUMP; i++)
		for(int b = 0; b < 64; b++) {
			if (JUMP[i] & UINT64_C(1) << b) {
				s0 ^= s[0];
				s1 ^= s[1];
			}
			xoroshiro128pp_next(s);
		}

	s[0] = s0;
	s[1] = s1;
}

void xoroshiro128pp_long_jump(uint64_t (&s)[2]) {
	static const uint64_t LONG_JUMP[] = { 0x360fd5f2cf8d5d99, 0x9c6e6877736c46e3 };

	uint64_t s0 = 0;
	uint64_t s1 = 0;
	for(int i = 0; i < sizeof LONG_JUMP / sizeof *LONG_JUMP; i++)
		for(int b = 0; b < 64; b++) {
			if (LONG_JUMP[i] & UINT64_C(1) << b) {
				s0 ^= s[0];
				s1 ^= s[1];
			}
			xoroshiro128pp_next(s);
		}

	s[0] = s0;
	s[1] = s1;
}

// https://prng.di.unimi.it/xoshiro256plusplus.c
uint64_t xoshiro256pp_next(uint64_t (&s)[4]) {
	const uint64_t result = rotl(s[0] + s[3], 23) + s[0];

	const uint64_t t = s[1] << 17;

	s[2] ^= s[0];
	s[3] ^= s[1];
	s[1] ^= s[2];
	s[0] ^= s[3];

	s[2] ^= t;

	s[3] = rotl(s[3], 45);

	return result;
}

void xoshiro256pp_jump(uint64_t (&s)[4]) {
	static const uint64_t JUMP[] = { 0x180ec6d33cfd0aba, 0xd5a61266f0c9392c, 0xa9582618e03fc9aa, 0x39abdc4529b1661c };

	uint64_t s0 = 0;
	uint64_t s1 = 0;
	uint64_t s2 = 0;
	uint64_t s3 = 0;
	for(int i = 0; i < sizeof JUMP / sizeof *JUMP; i++)
		for(int b = 0; b < 64; b++) {
			if (JUMP[i] & UINT64_C(1) << b) {
				s0 ^= s[0];
				s1 ^= s[1];
				s2 ^= s[2];
				s3 ^= s[3];
			}
			xoshiro256pp_next(s);
		}

	s[0] = s0;
	s[1] = s1;
	s[2] = s2;
	s[3] = s3;
}

void xoshiro256pp_long_jump(uint64_t (&s)[4]) {
	static const uint64_t LONG_JUMP[] = { 0x76e15d3efefdcbbf, 0xc5004e441c522fb3, 0x77710069854ee241, 0x39109bb02acbe635 };

	uint64_t s0 = 0;
	uint64_t s1 = 0;
	uint64_t s2 = 0;
	uint64_t s3 = 0;
	for(int i = 0; i < sizeof LONG_JUMP / sizeof *LONG_JUMP; i++)
		for(int b = 0; b < 64; b++) {
			if (LONG_JUMP[i] & UINT64_C(1) << b) {
				s0 ^= s[0];
				s1 ^= s[1];
				s2 ^= s[2];
				s3 ^= s[3];
			}
			xoshiro256pp_next(s);
		}

	s[0] = s0;
	s[1] = s1;
	s[2] = s2;
	s[3] = s3;
}

// https://prng.di.unimi.it/xoshiro256starstar.c
uint64_t xoshiro256ss_next(uint64_t (&s)[4]) {
	const uint64_t result = rotl(s[1] * 5, 7) * 9;

	const uint64_t t = s[1] << 17;

	s[2] ^= s[0];
	s[3] ^= s[1];
	s[1] ^= s[2];
	s[0] ^= s[3];

	s[2] ^= t;

	s[3] = rotl(s[3], 45);

	return result;
}

void xoshiro256ss_jump(uint64_t (&s)[4]) {
	static const uint64_t JUMP[] = { 0x180ec6d33cfd0aba, 0xd5a61266f0c9392c, 0xa9582618e03fc9aa, 0x39abdc4529b1661c };

	uint64_t s0 = 0;
	uint64_t s1 = 0;
	uint64_t s2 = 0;
	uint64_t s3 = 0;
	for(int i = 0; i < sizeof JUMP / sizeof *JUMP; i++)
		for(int b = 0; b < 64; b++) {
			if (JUMP[i] & UINT64_C(1) << b) {
				s0 ^= s[0];
				s1 ^= s[1];
				s2 ^= s[2];
				s3 ^= s[3];
			}
			xoshiro256ss_next(s);
		}

	s[0] = s0;
	s[1] = s1;
	s[2] = s2;
	s[3] = s3;
}

void xoshiro256ss_long_jump(uint64_t (&s)[4]) {
	static const uint64_t LONG_JUMP[] = { 0x76e15d3efefdcbbf, 0xc5004e441c522fb3, 0x77710069854ee241, 0x39109bb02acbe635 };

	uint64_t s0 = 0;
	uint64_t s1 = 0;
	uint64_t s2 = 0;
	uint64_t s3 = 0;
	for(int i = 0; i < sizeof LONG_JUMP / sizeof *LONG_JUMP; i++)
		for(int b = 0; b < 64; b++) {
			if (LONG_JUMP[i] & UINT64_C(1) << b) {
				s0 ^= s[0];
				s1 ^= s[1];
				s2 ^= s[2];
				s3 ^= s[3];
			}
			xoshiro256ss_next(s);
		}

	s[0] = s0;
	s[1] = s1;
	s[2] = s2;
	s[3] = s3;
}

template<typename T>
struct GolangName;

template<> struct GolangName<uint8_t> { inline static constexpr const char* name = "uint8"; };
template<> struct GolangName<uint16_t> { inline static constexpr const char* name = "uint16"; };
template<> struct GolangName<uint32_t> { inline static constexpr const char* name = "uint32"; };
template<> struct GolangName<uint64_t> { inline static constexpr const char* name = "uint64"; };

static inline void print(uint8_t v) { fmt::print("{:02X}", v); }
static inline void print(uint16_t v) { fmt::print("{:04X}", v); }
static inline void print(uint32_t v) { fmt::print("{:08X}", v); }
static inline void print(uint64_t v) { fmt::print("{:016X}", v); }
static inline void print(__uint128_t v) {
    print(v & 0xFFFF'FFFF'FFFF'FFFFull);
    print(v >> 64);
}

template<typename T>
static inline void print(std::span<T> span) {
    for (bool first = true; auto v : span) {
        if (!first)
            fmt::print(", ");
        first = false;
        fmt::print("0x");

        print(v);
    }
}


template<typename T, size_t N>
static inline void print(T (&arr)[N]) {
    return print(std::span<T>(arr));
}

template<typename T, typename U, size_t N>
static inline void gen_next_test(T (*next_fn)(U(&)[N])) {
    std::mt19937_64 engine{s_rd()};
    std::uniform_int_distribution<T> dist {};

    for (int i = 0; i < 16; i++) {
        U state[N];
        for (auto& v : state) v = dist(engine);

        fmt::print("{{[{}]{}{{", N, GolangName<U>::name);
        print(state);
        fmt::print("}}, []{}{{", GolangName<T>::name);

        for (int j = 0, first = true; j < 16; j++) {
            if (!first)
                fmt::print(", ");
            first = false;

            const auto res = next_fn(state);

            fmt::print("0x");
            print(res);
        }

        fmt::print("}}}},\n");
    }
}

template<typename T, size_t N>
static inline void gen_jump_test(void (*short_jumper)(T(&)[N]), void (*long_jumper)(T(&)[N])) {
    std::mt19937_64 engine{s_rd()};
    std::uniform_int_distribution<T> dist {};

    for (int i = 0; i < 8; i++) {
        T state_short[N];
        T state_long[N];
        for (auto& v : state_short) v = dist(engine);
        std::copy(state_short, state_short + N, state_long);

        fmt::print("{{[{}]{}{{", N, GolangName<T>::name);
        print(state_short);
        fmt::print("}}, [][2][{}]{}{{", N, GolangName<T>::name);

        for (int j = 0, first = true; j < 8; j++) {
            if (!first) fmt::print(", ");
            //else fmt::print("\n");
            first = false;

            short_jumper(state_short);
            short_jumper(state_long);

            fmt::print("{{");

            fmt::print("{{");
            print(state_short);
            fmt::print("}}, ");
            fmt::print("{{");
            print(state_long);
            fmt::print("}}");

            fmt::print("}}");
        }

        fmt::print("}}}},\n", N, GolangName<T>::name);
    }
}

#define GENTEST(_name) \
    fmt::print(#_name ": \n"); \
    gen_next_test(_name##_next);

#define GENJTEST(_name) \
    fmt::print(#_name ": \n"); \
    gen_jump_test(_name##_jump, _name##_long_jump);

int main() {
    //GENTEST(xoroshiro128pp);
}