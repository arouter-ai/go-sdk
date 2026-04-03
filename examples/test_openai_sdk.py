#!/usr/bin/env python3
"""
ARouter transparent proxy test using the standard OpenAI SDK.

Demonstrates that any OpenAI-compatible SDK
can connect to arouter by simply swapping base_url and api_key.

Supports testing with both main keys (lr_live_) and subkeys (lr_sub_).
API key and subkey behave identically — subkeys enable independent billing.

Usage:
    python test_openai_sdk.py                         # test all keys
    python test_openai_sdk.py --key lr_live_xxx       # test specific key
    python test_openai_sdk.py --subkeys-only          # test subkeys only
"""

import argparse
import os
import sys
import time
from openai import OpenAI

AROUTER_BASE_URL = os.getenv("AROUTER_BASE_URL", "http://localhost:18080/v1")

MAIN_KEY = "lr_live_61d1517a55a7c664a8717300ec6cddb0460aee2a502b17f5"

SUBKEYS = {
    "App-1 SubKey":       "lr_sub_e23ff90e75496b72517d179389f9fd8b9d9a4ee941259fe3",
    "Backend Service":    "lr_sub_fe1e15dc12954aa46940aa5c281e4515363271793dbf1b35",
    "Frontend Widget":    "lr_sub_f127ffc8a8a56f3dc7b5a6e636174e58d2530131fdb1c6a7",
}

MODELS = [
    "anthropic/claude-sonnet-4",
    "anthropic/claude-3.5-sonnet",
    "google/gemini-2.0-flash-001",
    "openai/gpt-4o-mini",
]


def make_client(api_key: str) -> OpenAI:
    return OpenAI(base_url=AROUTER_BASE_URL, api_key=api_key)


def test_chat(client: OpenAI, model: str) -> bool:
    start = time.time()
    try:
        resp = client.chat.completions.create(
            model=model,
            messages=[{"role": "user", "content": "Say hello in exactly 5 words."}],
            max_tokens=50,
        )
        elapsed = time.time() - start
        content = resp.choices[0].message.content
        usage = resp.usage
        print(f"    Response: {content}")
        print(f"    Tokens:   in={usage.prompt_tokens} out={usage.completion_tokens}")
        print(f"    Latency:  {elapsed:.2f}s  OK")
        return True
    except Exception as e:
        print(f"    FAIL ({time.time() - start:.2f}s): {e}")
        return False


def test_stream(client: OpenAI, model: str) -> bool:
    start = time.time()
    try:
        stream = client.chat.completions.create(
            model=model,
            messages=[{"role": "user", "content": "Count from 1 to 5, one per line."}],
            max_tokens=100,
            stream=True,
        )
        print("    Stream: ", end="", flush=True)
        for chunk in stream:
            if chunk.choices and chunk.choices[0].delta.content:
                print(chunk.choices[0].delta.content, end="", flush=True)
        elapsed = time.time() - start
        print(f"\n    Latency:  {elapsed:.2f}s  OK")
        return True
    except Exception as e:
        print(f"\n    FAIL ({time.time() - start:.2f}s): {e}")
        return False


def run_tests_for_key(label: str, api_key: str, models: list[str]) -> dict[str, bool]:
    key_type = "SubKey" if api_key.startswith("lr_sub_") else "MainKey"
    print(f"\n{'#'*60}")
    print(f"  {key_type}: {label}")
    print(f"  Key:     {api_key[:20]}...")
    print(f"{'#'*60}")

    client = make_client(api_key)
    results = {}

    for model in models:
        short = model.split("/", 1)[1]
        print(f"\n  [Chat] {short}")
        results[f"chat:{short}"] = test_chat(client, model)

    stream_model = models[0]
    short = stream_model.split("/", 1)[1]
    print(f"\n  [Stream] {short}")
    results[f"stream:{short}"] = test_stream(client, stream_model)

    return results


def main():
    parser = argparse.ArgumentParser(description="ARouter OpenAI SDK test")
    parser.add_argument("--key", help="Test a specific key only")
    parser.add_argument("--subkeys-only", action="store_true", help="Only test subkeys")
    parser.add_argument("--model", help="Test a specific model only")
    args = parser.parse_args()

    models = [args.model] if args.model else MODELS

    print("=" * 60)
    print("ARouter - Main Key & SubKey Comparison Test")
    print(f"Base URL: {AROUTER_BASE_URL}")
    print("=" * 60)

    all_results = {}

    if args.key:
        label = "Custom"
        for name, sk in SUBKEYS.items():
            if sk == args.key:
                label = name
                break
        if args.key == MAIN_KEY:
            label = "Main Key"
        all_results[label] = run_tests_for_key(label, args.key, models)
    else:
        if not args.subkeys_only:
            all_results["Main Key"] = run_tests_for_key("Main Key", MAIN_KEY, models)

        for name, subkey in SUBKEYS.items():
            all_results[name] = run_tests_for_key(name, subkey, models)

    # Summary
    print(f"\n{'='*60}")
    print("Final Summary")
    print(f"{'='*60}")

    total_pass, total_all = 0, 0
    for label, results in all_results.items():
        passed = sum(1 for v in results.values() if v)
        total = len(results)
        total_pass += passed
        total_all += total
        key_type = "lr_sub_" if label != "Main Key" else "lr_live_"
        status = "ALL PASS" if passed == total else f"{passed}/{total}"
        print(f"  [{status}] {label} ({key_type}...)")
        for name, ok in results.items():
            mark = "PASS" if ok else "FAIL"
            print(f"         [{mark}] {name}")

    print(f"\n  Total: {total_pass}/{total_all} passed")
    print()
    if total_pass == total_all:
        print("  Main key and subkeys behave identically.")
        print("  SubKeys enable independent billing & usage tracking.")
    else:
        sys.exit(1)


if __name__ == "__main__":
    main()
