#include <iostream>

int main() {
    long long N = 1e14;
    long long sum = 0;
    for (int i = 1; i <= N; i++) {
        sum += i;
    }
    std::cout << sum << "\n";
}