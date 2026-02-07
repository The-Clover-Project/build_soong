# Copyright (C) 2026 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import unittest
import json
import io
import dataclasses
from typing import Callable, Optional
from unittest.mock import patch

from interface.server import MCPServer
# Needed to register the TOOLS
import interface.defs
from api.env import BuildContext

class TestMCPServer(unittest.TestCase):
    def test_server_loop(self) -> None:
        # Simulate input stream: initialize -> tools/list
        input_data = (
            '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}\n'
            '{"jsonrpc": "2.0", "method": "notifications/initialized"}\n'
            '{"jsonrpc": "2.0", "id": 2, "method": "tools/list"}\n'
        )

        mock_stdin = io.StringIO(input_data)
        mock_stdout = io.StringIO()

        # Patch sys.stdin and sys.stdout
        with patch('sys.stdin', mock_stdin), patch('sys.stdout', mock_stdout):
            server = MCPServer()
            try:
                server.run()
            except SystemExit:
                pass

        # Analyze output
        output_lines = mock_stdout.getvalue().strip().split('\n')
        self.assertTrue(len(output_lines) >= 2, "Should have at least 2 responses")

        # Check Response 1 (initialize)
        resp1 = json.loads(output_lines[0])
        self.assertEqual(resp1.get("jsonrpc"), "2.0")
        self.assertEqual(resp1.get("id"), 1)
        self.assertIn("capabilities", resp1.get("result", {}))

        # Check Response 2 (tools/list)
        resp2 = json.loads(output_lines[1])
        self.assertEqual(resp2.get("jsonrpc"), "2.0")
        self.assertEqual(resp2.get("id"), 2)
        tools = resp2.get("result", {}).get("tools", [])
        self.assertTrue(len(tools) > 0, "Should list tools")

    def test_batch_request(self) -> None:
        # Batch: [ping, tools/list]
        input_data = '[{"jsonrpc": "2.0", "id": 1, "method": "ping"}, {"jsonrpc": "2.0", "id": 2, "method": "tools/list"}]\n'

        mock_stdin = io.StringIO(input_data)
        mock_stdout = io.StringIO()

        with patch('sys.stdin', mock_stdin), patch('sys.stdout', mock_stdout):
            server = MCPServer()
            try:
                server.run()
            except SystemExit:
                pass

        output = mock_stdout.getvalue().strip()
        response = json.loads(output)

        self.assertIsInstance(response, list)
        self.assertEqual(len(response), 2)

        # Order is preserved in processing but response order in JSON-RPC batch is not guaranteed by spec,
        # but our implementation likely preserves order of processing.
        # Let's find by ID.
        resp1 = next(r for r in response if r['id'] == 1)
        resp2 = next(r for r in response if r['id'] == 2)

        self.assertEqual(resp1['result'], {})
        self.assertIn("tools", resp2['result'])

    def test_lifecycle(self) -> None:
        # initialize -> shutdown -> exit
        input_data = (
            '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}\n'
            '{"jsonrpc": "2.0", "id": 2, "method": "shutdown"}\n'
            '{"jsonrpc": "2.0", "method": "exit"}\n'
        )

        mock_stdin = io.StringIO(input_data)
        mock_stdout = io.StringIO()

        with patch('sys.stdin', mock_stdin), patch('sys.stdout', mock_stdout):
            server = MCPServer()
            # exit will raise SystemExit
            with self.assertRaises(SystemExit) as cm:
                server.run()
            self.assertEqual(cm.exception.code, 0)

    def test_progress(self) -> None:
        # Mock a tool in TOOLS that calls the progress callback
        # We need to temporarily modify TOOLS
        from interface.registry import TOOLS, ToolDefinition
        from interface.schema import ToolArgs

        # Define a dummy tool
        @dataclasses.dataclass(frozen=True)
        class ProgressArgs(ToolArgs):
            product: str = "p"
            release: str = "r"
            variant: str = "v"

        def progress_tool(ctx: BuildContext, args: ProgressArgs, progress_callback: Optional[Callable[[float, Optional[float]], None]] = None) -> None:
            if progress_callback:
                progress_callback(50.0, 100.0)

        # Inject into TOOLS
        original_tools = TOOLS.copy()
        TOOLS["progress_tool"] = ToolDefinition("progress_tool", ProgressArgs, progress_tool)

        try:
            # Request with progressToken
            input_data = (
                '{"jsonrpc": "2.0", "id": 1, "method": "tools/call", '
                '"params": {"name": "progress_tool", "arguments": {}, "progressToken": "pt-123"}}\n'
            )

            mock_stdin = io.StringIO(input_data)
            mock_stdout = io.StringIO()

            with patch('sys.stdin', mock_stdin), patch('sys.stdout', mock_stdout):
                server = MCPServer()
                # Run one loop iteration effectively
                # server.run() loops until EOF
                server.run()

            output_lines = mock_stdout.getvalue().strip().split('\n')

            # Expecting:
            # 1. Progress notification
            # 2. Result

            # Find progress notification
            progress_notif = None
            for line in output_lines:
                msg = json.loads(line)
                if msg.get("method") == "notifications/progress":
                    progress_notif = msg
                    break

            self.assertIsNotNone(progress_notif)
            if progress_notif:
                self.assertEqual(progress_notif["params"]["progressToken"], "pt-123")
                self.assertEqual(progress_notif["params"]["progress"], 50.0)
                self.assertEqual(progress_notif["params"]["total"], 100.0)

        finally:
            # Restore TOOLS
            TOOLS.clear()
            TOOLS.update(original_tools)

if __name__ == "__main__":
    unittest.main()
