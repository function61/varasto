import { reloadCurrentPage } from 'f61ui/browserutils';
import { DangerAlert } from 'f61ui/component/alerts';
import { Glyphicon } from 'f61ui/component/bootstrap';
import { Loading } from 'f61ui/component/loading';
import { httpMustBeOk, makeQueryParams } from 'f61ui/httputil';
import { dateObjToDateTime } from 'f61ui/types';
import { unrecognizedValue, shouldAlwaysSucceed } from 'f61ui/utils';
import {
	commitChangeset,
	generateIds,
	uploadFileUrl,
} from 'generated/stoserver/stoservertypes_endpoints';
import {
	File as File2, // conflicts with HTML's "File" interface
} from 'generated/stoserver/stoservertypes_types';
import * as React from 'react';

enum uploadProgressState {
	Queued,
	Uploading,
	Succeeded,
	Errored,
}

interface UploadProgress {
	file: File;
	state: uploadProgressState; // can't monitor upload progress
}

interface FileUploadAreaProps {
	collectionId: string;
	collectionRevision: string;
}

interface FileUploadAreaState {
	uploads: UploadProgress[];
}

export class FileUploadArea extends React.Component<FileUploadAreaProps, FileUploadAreaState> {
	state: FileUploadAreaState = {
		uploads: [],
	};

	render() {
		return (
			<div>
				<table className="table table-striped table-hover">
					<tbody>
						{this.state.uploads.map((upload) => (
							<tr>
								<td style={{ width: '1%' }}>{upload.file.name}</td>
								<td>{this.stateToNode(upload.state)}</td>
							</tr>
						))}
					</tbody>
				</table>

				<input
					type="file"
					id="upload"
					multiple={true}
					onChange={(e) => {
						this.filesForUploadSelected(e);
					}}
				/>
			</div>
		);
	}

	private stateToNode(state: uploadProgressState): React.ReactNode {
		switch (state) {
			case uploadProgressState.Queued:
				return <Glyphicon icon="time" />;
			case uploadProgressState.Uploading:
				return <Loading />;
			case uploadProgressState.Succeeded:
				return <Glyphicon icon="ok" />;
			case uploadProgressState.Errored:
				return <DangerAlert>errored</DangerAlert>;
			default:
				throw unrecognizedValue(state);
		}
	}

	private filesForUploadSelected(e: React.ChangeEvent<HTMLInputElement>) {
		if (!e.target.files || e.target.files.length === 0) {
			return;
		}

		// is not an array, so convert to one
		const files: File[] = [];
		// tslint:disable-next-line:prefer-for-of
		for (let i = 0; i < e.target.files.length; ++i) {
			files.push(e.target.files[i]);
		}

		shouldAlwaysSucceed(this.uploadAllFiles(files));
	}

	private async uploadAllFiles(files: File[]) {
		const uploads = [];

		for (const file of files) {
			uploads.push({
				file,
				state: uploadProgressState.Queued,
			});
		}

		this.setState({ uploads });

		const createdFiles: File2[] = [];
		for (const upload of uploads) {
			upload.state = uploadProgressState.Uploading;

			this.setState({ uploads });

			createdFiles.push(await this.uploadOneFile(upload.file));

			upload.state = uploadProgressState.Succeeded;

			this.setState({ uploads });
		}

		await commitChangeset(this.props.collectionId, {
			ID: (await generateIds()).Changeset,
			Parent: this.props.collectionRevision,
			Created: dateObjToDateTime(new Date()),
			FilesCreated: createdFiles,
			FilesUpdated: [],
			FilesDeleted: [],
		});

		reloadCurrentPage();
	}

	private async uploadOneFile(file: File): Promise<File2> {
		const uploadEndpoint = makeQueryParams(uploadFileUrl(this.props.collectionId), {
			mtime: file.lastModified.toString(),
			filename: file.name, // spec says: "without path information"
		});

		// TODO: upload progress

		const response = await fetch(uploadEndpoint, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/octet-stream',
			},
			body: file,
		});

		await httpMustBeOk(response);

		return await response.json();
	}
}
