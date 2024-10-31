async function postData() {
  const res = await fetch("http://localhost:8080/submit", {
    method: "POST",
    body: JSON.stringify({
      cpp_code:
        "#include<iostream>\nint main() { int x; std::cin >> x; for (int i = 1; i<=1000000000; i++) continue; std::cout << x * 2; return 0; }",
      test_cases: ["2", "3", "5"],
    }),
  });
  const data = await res.json();
  console.log(data);
  return data;
}

for (let i = 1; i <= 10; i++) {
  postData();
}
