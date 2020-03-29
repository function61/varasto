import { Result } from 'f61ui/component/result';
import { DangerLabel, Glyphicon, Panel, SuccessLabel } from 'f61ui/component/bootstrap';
import { CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
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
import { SettingsLayout } from 'layout/settingslayout';
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
			<SettingsLayout title="Scheduled jobs" breadcrumbs={[]}>
				<Panel heading="System-level jobs">{this.renderSystemJobs()}</Panel>
				<Panel heading="User-level jobs">
					<p>TODO</p>
					<p>
						Why? Since we already have a fully-featured scheduler with great monitoring
						capabilities, it's only a minor effort to support docker run ... commands so
						that we can have meaningful user-level jobs that actually do something with
						the rich data platform that Varasto provides.
					</p>
				</Panel>
			</SettingsLayout>
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
					<div>
						<button
							className="btn btn-primary"
							type="button"
							data-toggle="collapse"
							data-target="#collapseExample"
							aria-expanded="false"
							aria-controls="collapseExample">
							Show {disabledJobs.length} disabled job(s)
						</button>
						<div id="collapseExample" className="collapse">
							{this.renderJobsTable(disabledJobs)}
						</div>
					</div>
				)}
			</div>
		);
	}

	private renderJobsTable(jobs: SchedulerJob[]) {
		return (
			<table className="table table-striped table-hover">
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
