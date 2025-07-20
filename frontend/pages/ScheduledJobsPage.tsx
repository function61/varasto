import {
	CollapsePanel,
	DangerLabel,
	Glyphicon,
	Panel,
	SuccessLabel,
	tableClassStripedHover,
} from 'f61ui/component/bootstrap';
import { CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Result } from 'f61ui/component/result';
import { Timestamp } from 'f61ui/component/timestamp';
import { formatDistance2 } from 'f61ui/utils';
import {
	ScheduledjobChangeSchedule,
	ScheduledjobDisable,
	ScheduledjobEnable,
	ScheduledjobStart,
} from 'generated/stoserver/stoservertypes_commands';
import { getSchedulerJobs } from 'generated/stoserver/stoservertypes_endpoints';
import { SchedulerJob } from 'generated/stoserver/stoservertypes_types';
import { AdminLayout } from 'layout/AdminLayout';
import * as React from 'react';

interface ScheduledJobsPageState {
	schedulerJobs: Result<SchedulerJob[]>;
}

export default class ScheduledJobsPage extends React.Component<{}, ScheduledJobsPageState> {
	state: ScheduledJobsPageState = {
		schedulerJobs: new Result<SchedulerJob[]>((_) => {
			this.setState({ schedulerJobs: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		return (
			<AdminLayout title="Scheduled jobs" breadcrumbs={[]}>
				<Panel heading="System-level jobs">{this.renderSystemJobs()}</Panel>
				<CollapsePanel heading="User-level jobs">
					<p>TODO</p>
					<p>
						Why? Since we already have a fully-featured scheduler with great monitoring
						capabilities, it's only a minor effort to support docker run ... commands so
						that we can have meaningful user-level jobs that actually do something with
						the rich data platform that Varasto provides.
					</p>
				</CollapsePanel>
			</AdminLayout>
		);
	}

	private renderSystemJobs() {
		const [jobs, loadingOrError] = this.state.schedulerJobs.unwrap();

		const mustJobs = jobs || [];

		const enabledJobs = mustJobs.filter((j) => j.Enabled);
		const disabledJobs = mustJobs.filter((j) => !j.Enabled);

		return (
			<div>
				{loadingOrError}

				{this.renderJobsTable(enabledJobs)}

				{disabledJobs.length > 0 && (
					<CollapsePanel
						heading={`Show ${disabledJobs.length} disabled job(s)`}
						visualStyle="warning">
						{this.renderJobsTable(disabledJobs)}
					</CollapsePanel>
				)}
			</div>
		);
	}

	private renderJobsTable(jobs: SchedulerJob[]) {
		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th>Job</th>
						<th>Schedule</th>
						<th>Next</th>
						<th>Previous</th>
						<th>Runtime</th>
						<th />
					</tr>
				</thead>
				<tbody>
					{jobs.map((job) => {
						const nextRun = job.NextRun;
						const lastRun = job.LastRun;

						return (
							<tr key={job.Id}>
								<td>
									{job.Description}
									&nbsp;
									{job.Running && (
										<SuccessLabel>
											<Glyphicon icon="off" />
											running
										</SuccessLabel>
									)}
								</td>
								<td>{job.Schedule}</td>
								<td>{nextRun && <Timestamp ts={nextRun} />}</td>
								<td>
									{lastRun && (
										<div>
											{lastRun.Error ? (
												<DangerLabel>{lastRun.Error}</DangerLabel>
											) : (
												<SuccessLabel>OK</SuccessLabel>
											)}
											&nbsp;
											<Timestamp ts={lastRun.Started} />
										</div>
									)}
								</td>
								<td>
									{lastRun && formatDistance2(lastRun.Started, lastRun.Finished)}
								</td>
								<td>
									<Dropdown>
										<CommandLink
											command={ScheduledjobStart(job.Id, {
												disambiguation: job.Description,
											})}
										/>
										<CommandLink
											command={ScheduledjobChangeSchedule(
												job.Id,
												job.Schedule,
												{ disambiguation: job.Description },
											)}
										/>
										<CommandLink
											command={ScheduledjobEnable(job.Id, {
												disambiguation: job.Description,
											})}
										/>
										<CommandLink
											command={ScheduledjobDisable(job.Id, {
												disambiguation: job.Description,
											})}
										/>
									</Dropdown>
								</td>
							</tr>
						);
					})}
				</tbody>
			</table>
		);
	}

	private fetchData() {
		this.state.schedulerJobs.load(() => getSchedulerJobs());
	}
}
