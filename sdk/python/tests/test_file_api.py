from __future__ import annotations

from pathlib import Path


def test_upload_and_download_metadata(managed_daemon, tmp_path: Path):
    client = managed_daemon["client_a"]
    upload_source = tmp_path / "upload.txt"
    upload_source.write_text("upload payload\n", encoding="utf-8")
    remote_path = tmp_path / "remote-upload.txt"

    upload = client.upload_file(str(upload_source), str(remote_path))
    assert upload.transfer_id

    upload_meta = client.get_execution(upload.transfer_id)
    assert upload_meta.last_upload_local_path == str(upload_source)
    assert upload_meta.last_upload_remote_path == str(remote_path)
    assert upload_meta.last_upload_transfer_id == upload.transfer_id

    download_target = tmp_path / "downloaded.txt"
    download = client.download_file(str(remote_path), str(download_target))
    assert download.transfer_id
    assert download_target.read_text(encoding="utf-8") == "upload payload\n"

    download_meta = client.get_execution(download.transfer_id)
    assert download_meta.last_download_local_path == str(download_target)
    assert download_meta.last_download_remote_path == str(remote_path)
    assert download_meta.last_download_transfer_id == download.transfer_id


def test_async_upload_and_download_metadata(managed_daemon, tmp_path: Path):
    client = managed_daemon["client_a"]
    upload_source = tmp_path / "upload-async.txt"
    upload_source.write_text("upload async payload\n", encoding="utf-8")
    remote_path = tmp_path / "remote-upload-async.txt"

    upload = client.upload_file_async(str(upload_source), str(remote_path)).result(timeout=10)
    assert upload.transfer_id

    upload_meta = client.get_execution(upload.transfer_id)
    assert upload_meta.last_upload_local_path == str(upload_source)
    assert upload_meta.last_upload_remote_path == str(remote_path)
    assert upload_meta.last_upload_transfer_id == upload.transfer_id

    download_target = tmp_path / "downloaded-async.txt"
    download = client.download_file_async(str(remote_path), str(download_target)).result(timeout=10)
    assert download.transfer_id
    assert download_target.read_text(encoding="utf-8") == "upload async payload\n"

    download_meta = client.get_execution(download.transfer_id)
    assert download_meta.last_download_local_path == str(download_target)
    assert download_meta.last_download_remote_path == str(remote_path)
    assert download_meta.last_download_transfer_id == download.transfer_id


def test_download_archive(managed_daemon, tmp_path: Path):
    client = managed_daemon["client_a"]
    archive_source = tmp_path / "archive-source"
    archive_source.mkdir()
    (archive_source / "one.txt").write_text("one", encoding="utf-8")
    (archive_source / "two.txt").write_text("two", encoding="utf-8")

    archive_target = tmp_path / "bundle.zip"
    download = client.download_archive([str(archive_source)], str(archive_target))

    assert download.transfer_id
    assert archive_target.exists()
    assert archive_target.stat().st_size > 0

    archive_meta = client.get_execution(download.transfer_id)
    assert archive_meta.last_download_local_path == str(archive_target)
    assert str(archive_source) in archive_meta.last_download_remote_path


def test_async_download_archive(managed_daemon, tmp_path: Path):
    client = managed_daemon["client_a"]
    archive_source = tmp_path / "archive-source-async"
    archive_source.mkdir()
    (archive_source / "one.txt").write_text("one", encoding="utf-8")
    (archive_source / "two.txt").write_text("two", encoding="utf-8")

    archive_target = tmp_path / "bundle-async.zip"
    download = client.download_archive_async([str(archive_source)], str(archive_target)).result(timeout=10)

    assert download.transfer_id
    assert archive_target.exists()
    assert archive_target.stat().st_size > 0

    archive_meta = client.get_execution(download.transfer_id)
    assert archive_meta.last_download_local_path == str(archive_target)
    assert str(archive_source) in archive_meta.last_download_remote_path
