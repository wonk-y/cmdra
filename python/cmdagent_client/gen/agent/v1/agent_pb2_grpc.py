"""Manual gRPC bindings for agent.v1.AgentService.

This file mirrors the small subset that grpcio-tools would generate so the
Python SDK can be rebuilt without needing the code generator installed.
"""

import grpc

from . import agent_pb2 as agent__pb2


class AgentServiceStub:
    """Client stub for the cmdagent AgentService."""

    def __init__(self, channel: grpc.Channel) -> None:
        self.StartCommand = channel.unary_unary(
            "/agent.v1.AgentService/StartCommand",
            request_serializer=agent__pb2.StartCommandRequest.SerializeToString,
            response_deserializer=agent__pb2.StartCommandResponse.FromString,
        )
        self.StartShell = channel.unary_unary(
            "/agent.v1.AgentService/StartShell",
            request_serializer=agent__pb2.StartShellRequest.SerializeToString,
            response_deserializer=agent__pb2.StartShellResponse.FromString,
        )
        self.GetExecution = channel.unary_unary(
            "/agent.v1.AgentService/GetExecution",
            request_serializer=agent__pb2.GetExecutionRequest.SerializeToString,
            response_deserializer=agent__pb2.GetExecutionResponse.FromString,
        )
        self.ListExecutions = channel.unary_unary(
            "/agent.v1.AgentService/ListExecutions",
            request_serializer=agent__pb2.ListExecutionsRequest.SerializeToString,
            response_deserializer=agent__pb2.ListExecutionsResponse.FromString,
        )
        self.CancelExecution = channel.unary_unary(
            "/agent.v1.AgentService/CancelExecution",
            request_serializer=agent__pb2.CancelExecutionRequest.SerializeToString,
            response_deserializer=agent__pb2.CancelExecutionResponse.FromString,
        )
        self.ReadOutput = channel.unary_stream(
            "/agent.v1.AgentService/ReadOutput",
            request_serializer=agent__pb2.ReadOutputRequest.SerializeToString,
            response_deserializer=agent__pb2.OutputChunk.FromString,
        )
        self.Attach = channel.stream_stream(
            "/agent.v1.AgentService/Attach",
            request_serializer=agent__pb2.AttachRequest.SerializeToString,
            response_deserializer=agent__pb2.AttachEvent.FromString,
        )
        self.UploadFile = channel.stream_unary(
            "/agent.v1.AgentService/UploadFile",
            request_serializer=agent__pb2.UploadFileRequest.SerializeToString,
            response_deserializer=agent__pb2.UploadFileResponse.FromString,
        )
        self.DownloadFile = channel.unary_stream(
            "/agent.v1.AgentService/DownloadFile",
            request_serializer=agent__pb2.DownloadFileRequest.SerializeToString,
            response_deserializer=agent__pb2.FileChunk.FromString,
        )
        self.DownloadArchive = channel.unary_stream(
            "/agent.v1.AgentService/DownloadArchive",
            request_serializer=agent__pb2.DownloadArchiveRequest.SerializeToString,
            response_deserializer=agent__pb2.FileChunk.FromString,
        )
