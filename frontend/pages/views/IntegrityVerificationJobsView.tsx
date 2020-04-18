import { thousandSeparate } from 'component/numberformatter';
import { RefreshButton } from 'component/refreshbutton';
import {
	DangerLabel,
	DefaultLabel,
	Glyphicon,
	SuccessLabel,
	WarningLabel,
	tableClassStripedHover,
} from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { ProgressBar } from 'f61ui/component/progressbar';
import { Timestamp } from 'f61ui/component/timestamp';
import { formatDistance2, unrecognizedValue } from 'f61ui/utils';
import {
	IntegrityverificationjobResume,
	IntegrityverificationjobStop,
	VolumeVerifyIntegrity,
} from 'generated/stoserver/stoservertypes_commands';
import {
	IntegrityVerificationJob,
	Volume,
	VolumeTechnology,
} from 'generated/stoserver/stoservertypes_types';
import * as React from 'react';

interface IntegirtyVerificationJobsViewProps {
	volumes: Volume[];
	jobs: IntegrityVerificationJob[];
	refresh: () => void;
}

interface IntegirtyVerificationJobsViewState {
	ivHistoricalJobsForVolumeUuid?: string;
}

export default class IntegirtyVerificationJobsView extends React.Component<
	IntegirtyVerificationJobsViewProps,
	IntegirtyVerificationJobsViewState
> {
	state: IntegirtyVerificationJobsViewState = {};

	render() {
		const rows: JSX.Element[] = [];

		for (const vol of this.props.volumes) {
			const jobs = this.props.jobs
				.filter((job) => job.VolumeId === vol.Id)
				.sort((a, b) => (a.Created < b.Created ? -1 : 1))
				.reverse();

			rows.push(this.row(vol, jobs[0], true));

			if (this.state.ivHistoricalJobsForVolumeUuid === vol.Uuid) {
				const historicalJobs = jobs.slice(1); // can be empty

				for (const historicalJob of historicalJobs) {
					rows.push(this.row(vol, historicalJob, false));
				}
			}
		}

		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th></th>
						<th>Volume</th>
						<th>
							{this.state.ivHistoricalJobsForVolumeUuid ? 'Scanned' : 'Last scan'}
						</th>
						<th>Runtime</th>
						<th>Size</th>
						<th></th>
						<th style={{ width: '1%' }} />
					</tr>
				</thead>
				<tbody>{rows}</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>
							{' '}
							<RefreshButton
								refresh={() => {
									this.props.refresh();
								}}
							/>
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private row(
		vol: Volume,
		job: IntegrityVerificationJob | undefined,
		showingHistorical: boolean,
	) {
		const rowKey = vol.Uuid + (job ? '-' + job.Id : '');

		if (!job) {
			return (
				<tr key={rowKey}>
					<td>{showingHistorical && volumeTechnologyBadge(vol.Technology)}</td>
					<td>{showingHistorical && vol.Label}</td>
					<td>
						<span className="text-muted">(Never scanned)</span>
					</td>
					<td></td>
					<td>Volume size {bytesToHumanReadable(vol.BlobSizeTotal)}</td>
					<td></td>
					<td>
						{' '}
						<Dropdown>
							<CommandLink
								command={VolumeVerifyIntegrity(vol.Id, {
									disambiguation: vol.Label,
								})}
							/>
						</Dropdown>
					</td>
				</tr>
			);
		}

		const completed = job.Completed;
		const nowOrCompleted: Date = completed ? new Date(completed) : new Date();
		const runtimeSeconds = (nowOrCompleted.getTime() - new Date(job.Created).getTime()) / 1000;
		const bytesPerSecond = runtimeSeconds > 0 ? job.BytesScanned / runtimeSeconds : 0;

		return (
			<tr key={rowKey}>
				<td title={job.Id}>{showingHistorical && volumeTechnologyBadge(vol.Technology)}</td>
				<td>{showingHistorical && vol.Label}</td>
				<td style={{ width: '25%' }}>{jobStatus(job)}</td>
				<td title={bytesToHumanReadable(bytesPerSecond) + '/s'}>
					{completed ? (
						formatDistance2(job.Created, completed)
					) : (
						<span>
							started <Timestamp ts={job.Created} />
						</span>
					)}
				</td>
				<td title={'Volume current size ' + bytesToHumanReadable(vol.BlobSizeTotal)}>
					Scanned {bytesToHumanReadable(job.BytesScanned)}
				</td>
				<td title={'Error count: ' + thousandSeparate(job.ErrorsFound)}>
					<Glyphicon icon="list-alt" title={job.Report} />
					&nbsp;
					{showingHistorical && (
						<Glyphicon
							icon="search"
							title="View historical jobs"
							click={() => {
								this.toggleHistoricalJobs(vol.Uuid);
							}}
						/>
					)}
				</td>
				<td>
					{completed && (
						<Dropdown>
							<CommandLink
								command={VolumeVerifyIntegrity(vol.Id, {
									disambiguation: vol.Label,
								})}
							/>
						</Dropdown>
					)}
					{!completed && (
						<Dropdown>
							<CommandLink command={IntegrityverificationjobResume(job.Id)} />
							<CommandLink command={IntegrityverificationjobStop(job.Id)} />
						</Dropdown>
					)}
				</td>
			</tr>
		);
	}

	private toggleHistoricalJobs(volUuid: string) {
		const currentlySelected = this.state.ivHistoricalJobsForVolumeUuid === volUuid;

		if (currentlySelected) {
			this.setState({ ivHistoricalJobsForVolumeUuid: undefined });
		} else {
			this.setState({ ivHistoricalJobsForVolumeUuid: volUuid });
		}
	}
}

/*
	stopped = !isCompleted AND !running
	running = !isCompleted AND running
	pass = isCompleted AND errors == 0
	fail = isCompleted AND errors > 0
*/
function jobStatus(job: IntegrityVerificationJob): React.ReactNode {
	const completed = job.Completed;

	const anyErrors = job.ErrorsFound > 0;

	if (completed === null) {
		if (!job.Running) {
			return <WarningLabel>Stopped</WarningLabel>;
		}

		// since the blobref is a SHA256, and its properties is uniform random distribution,
		// and since our b-tree based database table scans are alphabetical order, we
		// can deduce progress of scan by just looking at four first hexits:
		//
		// 0000 =>   0 %
		// 8000 =>  50 %
		// ffff => 100 %
		const lastCompletedBlobRefFourFirstHexits = job.LastCompletedBlobRef.substr(0, 4);

		const progress = (parseInt(lastCompletedBlobRefFourFirstHexits, 16) / 65535) * 100;

		return <ProgressBar progress={progress} colour={anyErrors ? 'danger' : undefined} />;
	}

	if (anyErrors) {
		return <DangerLabel>Failed</DangerLabel>;
	}

	return (
		<SuccessLabel>
			Pass <Timestamp ts={completed} />
		</SuccessLabel>
	);
}

export function volumeTechnologyBadge(tech: VolumeTechnology) {
	switch (tech) {
		case VolumeTechnology.DiskHdd:
			return <DefaultLabel>HDD</DefaultLabel>;
		case VolumeTechnology.DiskSsd:
			return <DefaultLabel>SSD</DefaultLabel>;
		case VolumeTechnology.Cloud:
			return <DefaultLabel>☁</DefaultLabel>;
		default:
			throw unrecognizedValue(tech);
	}
}
